/**
 * Cloudflare Worker Relay for ParentControl Guard
 * 提供无服务器公网控制端 API 与路由器指令中继调度
 */

export interface Env {
  PARENT_CONTROL_KV?: KVNamespace;
}

interface Command {
  id: string;
  type: 'LOCK' | 'UNLOCK' | 'BONUS' | 'SET_MEMBER' | 'DELETE_MEMBER' | 'UPDATE_SETTINGS' | 'ADD_APP' | 'DELETE_APP';
  member_id?: string;
  payload?: any;
  created_at: number;
}

interface RouterState {
  status: any;
  members: any[];
  devices: any[];
  categories: any[];
  settings: any;
  last_seen: number;
}

// 默认内存后备（在无 KV 绑定的纯本地环境下无缝降级）
const memoryStore = new Map<string, string>();

async function kvGet(env: Env, key: string): Promise<string | null> {
  if (env.PARENT_CONTROL_KV) {
    try {
      return await env.PARENT_CONTROL_KV.get(key);
    } catch (e) {
      console.warn('KV get failed, fallback to memory:', e);
    }
  }
  return memoryStore.get(key) || null;
}

async function kvPut(env: Env, key: string, value: string): Promise<void> {
  if (env.PARENT_CONTROL_KV) {
    try {
      await env.PARENT_CONTROL_KV.put(key, value);
      return;
    } catch (e) {
      console.warn('KV put failed, fallback to memory:', e);
    }
  }
  memoryStore.set(key, value);
}

function jsonResponse(data: any, status = 200, headers: Record<string, string> = {}): Response {
  return new Response(JSON.stringify(data), {
    status,
    headers: {
      'Content-Type': 'application/json; charset=utf-8',
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type, X-Pin-Code, Authorization, X-Router-Secret',
      ...headers,
    },
  });
}

export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const url = new URL(request.url);
    const path = url.pathname;
    const method = request.method;

    // 处理 CORS 预检请求
    if (method === 'OPTIONS') {
      return jsonResponse({ ok: true });
    }

    const routerKey = 'router:default:state';
    const queueKey = 'router:default:commands';

    // ==========================================
    // 1. 路由器通信专用接口 (Router Agent API)
    // ==========================================

    // 路由器上报全量状态并获取当前待办指令
    if (path === '/api/router/sync' && method === 'POST') {
      try {
        const body: any = await request.json();
        const state: RouterState = {
          status: body.status || {},
          members: body.members || [],
          devices: body.devices || [],
          categories: body.categories || [],
          settings: body.settings || {},
          last_seen: Date.now(),
        };
        await kvPut(env, routerKey, JSON.stringify(state));

        // 读取待办指令
        const queueRaw = await kvGet(env, queueKey);
        const commands: Command[] = queueRaw ? JSON.parse(queueRaw) : [];

        // 已经交给路由器，清空指令队列
        if (commands.length > 0) {
          await kvPut(env, queueKey, JSON.stringify([]));
        }

        return jsonResponse({
          success: true,
          commands: commands,
          server_time: new Date().toISOString(),
        });
      } catch (err: any) {
        return jsonResponse({ error: err.message }, 400);
      }
    }

    // 路由器长轮询等待新指令 (Long-Polling)
    if (path === '/api/router/poll' && method === 'GET') {
      const queueRaw = await kvGet(env, queueKey);
      const commands: Command[] = queueRaw ? JSON.parse(queueRaw) : [];
      if (commands.length > 0) {
        await kvPut(env, queueKey, JSON.stringify([]));
      }
      return jsonResponse({
        commands: commands,
        server_time: new Date().toISOString(),
      });
    }

    // ==========================================
    // 2. 状态读取与 App 鉴权
    // ==========================================

    // 获取当前状态
    const stateRaw = await kvGet(env, routerKey);
    const state: RouterState = stateRaw
      ? JSON.parse(stateRaw)
      : {
          status: { running: true, uptime_seconds: 0, total_devices: 0, active_devices: 0, managed_members: 0, kernel_dpi_ready: true, pin_required: false },
          members: [],
          devices: [],
          categories: [],
          settings: { enabled: true, pin_code: '' },
          last_seen: 0,
        };

    // 检查 PIN 码鉴权
    const configuredPin = state.settings?.pin_code || '';
    const pinRequired = configuredPin !== '';

    // 公开接口：状态查询
    if (path === '/api/status' && method === 'GET') {
      const isOnline = Date.now() - (state.last_seen || 0) < 60000;
      return jsonResponse({
        ...state.status,
        cloud_relay: true,
        router_online: isOnline,
        pin_required: pinRequired,
        server_time: new Date().toISOString(),
      });
    }

    // PIN 登录验证
    if (path === '/api/auth/login' && method === 'POST') {
      const body: any = await request.json();
      if (!pinRequired || body.pin === configuredPin) {
        return jsonResponse({ success: true, token: configuredPin });
      }
      return jsonResponse({ success: false, error: 'PIN 码错误' }, 401);
    }

    // 校验受保护路由的 PIN 码
    if (pinRequired) {
      const clientPin = request.headers.get('X-Pin-Code') || url.searchParams.get('pin') || '';
      if (clientPin !== configuredPin) {
        return jsonResponse({ error: 'PIN code required or invalid' }, 401);
      }
    }

    // 入队指令辅助函数
    async function enqueueCommand(cmd: Omit<Command, 'id' | 'created_at'>) {
      const queueRaw = await kvGet(env, queueKey);
      const queue: Command[] = queueRaw ? JSON.parse(queueRaw) : [];
      const newCmd: Command = {
        ...cmd,
        id: 'cmd_' + Date.now() + '_' + Math.random().toString(36).substring(2, 7),
        created_at: Date.now(),
      };
      queue.push(newCmd);
      await kvPut(env, queueKey, JSON.stringify(queue));
      return newCmd;
    }

    // ==========================================
    // 3. App 业务接口 (与路由器原生 API 完全兼容)
    // ==========================================

    // 成员列表与更新
    if (path === '/api/members' && method === 'GET') {
      return jsonResponse(state.members || []);
    }

    if (path === '/api/members' && method === 'POST') {
      const member: any = await request.json();
      if (!member.id) {
        member.id = 'm_' + Date.now();
      }

      // 乐观更新云端缓存
      const idx = state.members.findIndex((m) => m.id === member.id);
      if (idx >= 0) {
        state.members[idx] = member;
      } else {
        state.members.push(member);
      }
      await kvPut(env, routerKey, JSON.stringify(state));

      // 下发指令到路由器
      await enqueueCommand({ type: 'SET_MEMBER', member_id: member.id, payload: member });
      return jsonResponse(member);
    }

    // 成员动作
    if (path.startsWith('/api/members/')) {
      const parts = path.replace('/api/members/', '').split('/');
      const memberId = parts[0];

      // DELETE /api/members/:id
      if (method === 'DELETE' && parts.length === 1) {
        state.members = state.members.filter((m) => m.id !== memberId);
        await kvPut(env, routerKey, JSON.stringify(state));
        await enqueueCommand({ type: 'DELETE_MEMBER', member_id: memberId });
        return jsonResponse({ result: 'deleted' });
      }

      // POST /api/members/:id/lock
      if (method === 'POST' && parts[1] === 'lock') {
        const m = state.members.find((m) => m.id === memberId);
        if (m) m.is_locked = true;
        await kvPut(env, routerKey, JSON.stringify(state));
        await enqueueCommand({ type: 'LOCK', member_id: memberId });
        return jsonResponse({ status: 'locked' });
      }

      // POST /api/members/:id/unlock
      if (method === 'POST' && parts[1] === 'unlock') {
        const m = state.members.find((m) => m.id === memberId);
        if (m) m.is_locked = false;
        await kvPut(env, routerKey, JSON.stringify(state));
        await enqueueCommand({ type: 'UNLOCK', member_id: memberId });
        return jsonResponse({ status: 'unlocked' });
      }

      // POST /api/members/:id/bonus?minutes=30
      if (method === 'POST' && parts[1] === 'bonus') {
        const minutes = parseInt(url.searchParams.get('minutes') || '30');
        const m = state.members.find((m) => m.id === memberId);
        if (m) {
          m.bonus_until = new Date(Date.now() + minutes * 60000).toISOString();
        }
        await kvPut(env, routerKey, JSON.stringify(state));
        await enqueueCommand({ type: 'BONUS', member_id: memberId, payload: { minutes } });
        return jsonResponse({ status: 'bonus_applied', minutes });
      }
    }

    // 设备列表
    if (path === '/api/devices' && method === 'GET') {
      return jsonResponse(state.devices || []);
    }

    // 应用分类与特征
    if (path === '/api/apps' && method === 'GET') {
      return jsonResponse(state.categories || []);
    }

    if (path === '/api/apps' && method === 'POST') {
      const app: any = await request.json();
      await enqueueCommand({ type: 'ADD_APP', payload: app });
      return jsonResponse({ status: 'queued', app });
    }

    // 全局设置
    if (path === '/api/settings' && method === 'GET') {
      return jsonResponse(state.settings || {});
    }

    if (path === '/api/settings' && method === 'POST') {
      const newSettings: any = await request.json();
      state.settings = newSettings;
      await kvPut(env, routerKey, JSON.stringify(state));
      await enqueueCommand({ type: 'UPDATE_SETTINGS', payload: newSettings });
      return jsonResponse(newSettings);
    }

    return jsonResponse({ error: 'Endpoint not found', path }, 404);
  },
};
