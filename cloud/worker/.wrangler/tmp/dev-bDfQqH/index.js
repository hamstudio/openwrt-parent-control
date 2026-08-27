var __defProp = Object.defineProperty;
var __name = (target, value) => __defProp(target, "name", { value, configurable: true });

// .wrangler/tmp/bundle-efzei7/checked-fetch.js
var urls = /* @__PURE__ */ new Set();
function checkURL(request, init) {
  const url = request instanceof URL ? request : new URL(
    (typeof request === "string" ? new Request(request, init) : request).url
  );
  if (url.port && url.port !== "443" && url.protocol === "https:") {
    if (!urls.has(url.toString())) {
      urls.add(url.toString());
      console.warn(
        `WARNING: known issue with \`fetch()\` requests to custom HTTPS ports in published Workers:
 - ${url.toString()} - the custom port will be ignored when the Worker is published using the \`wrangler deploy\` command.
`
      );
    }
  }
}
__name(checkURL, "checkURL");
globalThis.fetch = new Proxy(globalThis.fetch, {
  apply(target, thisArg, argArray) {
    const [request, init] = argArray;
    checkURL(request, init);
    return Reflect.apply(target, thisArg, argArray);
  }
});

// src/index.ts
var memoryStore = /* @__PURE__ */ new Map();
async function kvGet(env, key) {
  if (env.PARENT_CONTROL_KV) {
    try {
      return await env.PARENT_CONTROL_KV.get(key);
    } catch (e) {
      console.warn("KV get failed, fallback to memory:", e);
    }
  }
  return memoryStore.get(key) || null;
}
__name(kvGet, "kvGet");
async function kvPut(env, key, value) {
  if (env.PARENT_CONTROL_KV) {
    try {
      await env.PARENT_CONTROL_KV.put(key, value);
      return;
    } catch (e) {
      console.warn("KV put failed, fallback to memory:", e);
    }
  }
  memoryStore.set(key, value);
}
__name(kvPut, "kvPut");
function jsonResponse(data, status = 200, headers = {}) {
  return new Response(JSON.stringify(data), {
    status,
    headers: {
      "Content-Type": "application/json; charset=utf-8",
      "Access-Control-Allow-Origin": "*",
      "Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
      "Access-Control-Allow-Headers": "Content-Type, X-Pin-Code, Authorization, X-Router-Secret",
      ...headers
    }
  });
}
__name(jsonResponse, "jsonResponse");
var src_default = {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);
    const path = url.pathname;
    const method = request.method;
    if (method === "OPTIONS") {
      return jsonResponse({ ok: true });
    }
    const routerKey = "router:default:state";
    const queueKey = "router:default:commands";
    if (path === "/api/router/sync" && method === "POST") {
      try {
        const body = await request.json();
        const state2 = {
          status: body.status || {},
          members: body.members || [],
          devices: body.devices || [],
          categories: body.categories || [],
          settings: body.settings || {},
          last_seen: Date.now()
        };
        await kvPut(env, routerKey, JSON.stringify(state2));
        const queueRaw = await kvGet(env, queueKey);
        const commands = queueRaw ? JSON.parse(queueRaw) : [];
        if (commands.length > 0) {
          await kvPut(env, queueKey, JSON.stringify([]));
        }
        return jsonResponse({
          success: true,
          commands,
          server_time: (/* @__PURE__ */ new Date()).toISOString()
        });
      } catch (err) {
        return jsonResponse({ error: err.message }, 400);
      }
    }
    if (path === "/api/router/poll" && method === "GET") {
      const queueRaw = await kvGet(env, queueKey);
      const commands = queueRaw ? JSON.parse(queueRaw) : [];
      if (commands.length > 0) {
        await kvPut(env, queueKey, JSON.stringify([]));
      }
      return jsonResponse({
        commands,
        server_time: (/* @__PURE__ */ new Date()).toISOString()
      });
    }
    const stateRaw = await kvGet(env, routerKey);
    const state = stateRaw ? JSON.parse(stateRaw) : {
      status: { running: true, uptime_seconds: 0, total_devices: 0, active_devices: 0, managed_members: 0, kernel_dpi_ready: true, pin_required: false },
      members: [],
      devices: [],
      categories: [],
      settings: { enabled: true, pin_code: "" },
      last_seen: 0
    };
    const configuredPin = state.settings?.pin_code || "";
    const pinRequired = configuredPin !== "";
    if (path === "/api/status" && method === "GET") {
      const isOnline = Date.now() - (state.last_seen || 0) < 6e4;
      return jsonResponse({
        ...state.status,
        cloud_relay: true,
        router_online: isOnline,
        pin_required: pinRequired,
        server_time: (/* @__PURE__ */ new Date()).toISOString()
      });
    }
    if (path === "/api/auth/login" && method === "POST") {
      const body = await request.json();
      if (!pinRequired || body.pin === configuredPin) {
        return jsonResponse({ success: true, token: configuredPin });
      }
      return jsonResponse({ success: false, error: "PIN \u7801\u9519\u8BEF" }, 401);
    }
    if (pinRequired) {
      const clientPin = request.headers.get("X-Pin-Code") || url.searchParams.get("pin") || "";
      if (clientPin !== configuredPin) {
        return jsonResponse({ error: "PIN code required or invalid" }, 401);
      }
    }
    async function enqueueCommand(cmd) {
      const queueRaw = await kvGet(env, queueKey);
      const queue = queueRaw ? JSON.parse(queueRaw) : [];
      const newCmd = {
        ...cmd,
        id: "cmd_" + Date.now() + "_" + Math.random().toString(36).substring(2, 7),
        created_at: Date.now()
      };
      queue.push(newCmd);
      await kvPut(env, queueKey, JSON.stringify(queue));
      return newCmd;
    }
    __name(enqueueCommand, "enqueueCommand");
    if (path === "/api/members" && method === "GET") {
      return jsonResponse(state.members || []);
    }
    if (path === "/api/members" && method === "POST") {
      const member = await request.json();
      if (!member.id) {
        member.id = "m_" + Date.now();
      }
      const idx = state.members.findIndex((m) => m.id === member.id);
      if (idx >= 0) {
        state.members[idx] = member;
      } else {
        state.members.push(member);
      }
      await kvPut(env, routerKey, JSON.stringify(state));
      await enqueueCommand({ type: "SET_MEMBER", member_id: member.id, payload: member });
      return jsonResponse(member);
    }
    if (path.startsWith("/api/members/")) {
      const parts = path.replace("/api/members/", "").split("/");
      const memberId = parts[0];
      if (method === "DELETE" && parts.length === 1) {
        state.members = state.members.filter((m) => m.id !== memberId);
        await kvPut(env, routerKey, JSON.stringify(state));
        await enqueueCommand({ type: "DELETE_MEMBER", member_id: memberId });
        return jsonResponse({ result: "deleted" });
      }
      if (method === "POST" && parts[1] === "lock") {
        const m = state.members.find((m2) => m2.id === memberId);
        if (m) m.is_locked = true;
        await kvPut(env, routerKey, JSON.stringify(state));
        await enqueueCommand({ type: "LOCK", member_id: memberId });
        return jsonResponse({ status: "locked" });
      }
      if (method === "POST" && parts[1] === "unlock") {
        const m = state.members.find((m2) => m2.id === memberId);
        if (m) m.is_locked = false;
        await kvPut(env, routerKey, JSON.stringify(state));
        await enqueueCommand({ type: "UNLOCK", member_id: memberId });
        return jsonResponse({ status: "unlocked" });
      }
      if (method === "POST" && parts[1] === "bonus") {
        const minutes = parseInt(url.searchParams.get("minutes") || "30");
        const m = state.members.find((m2) => m2.id === memberId);
        if (m) {
          m.bonus_until = new Date(Date.now() + minutes * 6e4).toISOString();
        }
        await kvPut(env, routerKey, JSON.stringify(state));
        await enqueueCommand({ type: "BONUS", member_id: memberId, payload: { minutes } });
        return jsonResponse({ status: "bonus_applied", minutes });
      }
    }
    if (path === "/api/devices" && method === "GET") {
      return jsonResponse(state.devices || []);
    }
    if (path === "/api/apps" && method === "GET") {
      return jsonResponse(state.categories || []);
    }
    if (path === "/api/apps" && method === "POST") {
      const app = await request.json();
      await enqueueCommand({ type: "ADD_APP", payload: app });
      return jsonResponse({ status: "queued", app });
    }
    if (path === "/api/settings" && method === "GET") {
      return jsonResponse(state.settings || {});
    }
    if (path === "/api/settings" && method === "POST") {
      const newSettings = await request.json();
      state.settings = newSettings;
      await kvPut(env, routerKey, JSON.stringify(state));
      await enqueueCommand({ type: "UPDATE_SETTINGS", payload: newSettings });
      return jsonResponse(newSettings);
    }
    return jsonResponse({ error: "Endpoint not found", path }, 404);
  }
};

// ../../../../../../../node_modules/wrangler/templates/middleware/middleware-ensure-req-body-drained.ts
var drainBody = /* @__PURE__ */ __name(async (request, env, _ctx, middlewareCtx) => {
  try {
    return await middlewareCtx.next(request, env);
  } finally {
    try {
      if (request.body !== null && !request.bodyUsed) {
        const reader = request.body.getReader();
        while (!(await reader.read()).done) {
        }
      }
    } catch (e) {
      console.error("Failed to drain the unused request body.", e);
    }
  }
}, "drainBody");
var middleware_ensure_req_body_drained_default = drainBody;

// ../../../../../../../node_modules/wrangler/templates/middleware/middleware-miniflare3-json-error.ts
function reduceError(e) {
  return {
    name: e?.name,
    message: e?.message ?? String(e),
    stack: e?.stack,
    cause: e?.cause === void 0 ? void 0 : reduceError(e.cause)
  };
}
__name(reduceError, "reduceError");
var jsonError = /* @__PURE__ */ __name(async (request, env, _ctx, middlewareCtx) => {
  try {
    return await middlewareCtx.next(request, env);
  } catch (e) {
    const error = reduceError(e);
    const body = JSON.stringify(error);
    const headers = {
      "Content-Type": "application/json",
      "MF-Experimental-Error-Stack": "true"
    };
    const encoded = encodeURIComponent(body);
    if (encoded.length <= 8192) {
      headers["MF-Experimental-Error-Stack-Payload"] = encoded;
    }
    return new Response(body, { status: 500, headers });
  }
}, "jsonError");
var middleware_miniflare3_json_error_default = jsonError;

// .wrangler/tmp/bundle-efzei7/middleware-insertion-facade.js
var __INTERNAL_WRANGLER_MIDDLEWARE__ = [
  middleware_ensure_req_body_drained_default,
  middleware_miniflare3_json_error_default
];
var middleware_insertion_facade_default = src_default;

// ../../../../../../../node_modules/wrangler/templates/middleware/common.ts
var __facade_middleware__ = [];
function __facade_register__(...args) {
  __facade_middleware__.push(...args.flat());
}
__name(__facade_register__, "__facade_register__");
function __facade_invokeChain__(request, env, ctx, dispatch, middlewareChain) {
  const [head, ...tail] = middlewareChain;
  const middlewareCtx = {
    dispatch,
    next(newRequest, newEnv) {
      return __facade_invokeChain__(newRequest, newEnv, ctx, dispatch, tail);
    }
  };
  return head(request, env, ctx, middlewareCtx);
}
__name(__facade_invokeChain__, "__facade_invokeChain__");
function __facade_invoke__(request, env, ctx, dispatch, finalMiddleware) {
  return __facade_invokeChain__(request, env, ctx, dispatch, [
    ...__facade_middleware__,
    finalMiddleware
  ]);
}
__name(__facade_invoke__, "__facade_invoke__");

// .wrangler/tmp/bundle-efzei7/middleware-loader.entry.ts
var __Facade_ScheduledController__ = class ___Facade_ScheduledController__ {
  constructor(scheduledTime, cron, noRetry) {
    this.scheduledTime = scheduledTime;
    this.cron = cron;
    this.#noRetry = noRetry;
  }
  scheduledTime;
  cron;
  static {
    __name(this, "__Facade_ScheduledController__");
  }
  #noRetry;
  noRetry() {
    if (!(this instanceof ___Facade_ScheduledController__)) {
      throw new TypeError("Illegal invocation");
    }
    this.#noRetry();
  }
};
function wrapExportedHandler(worker) {
  if (__INTERNAL_WRANGLER_MIDDLEWARE__ === void 0 || __INTERNAL_WRANGLER_MIDDLEWARE__.length === 0) {
    return worker;
  }
  for (const middleware of __INTERNAL_WRANGLER_MIDDLEWARE__) {
    __facade_register__(middleware);
  }
  const fetchDispatcher = /* @__PURE__ */ __name(function(request, env, ctx) {
    if (worker.fetch === void 0) {
      throw new Error("Handler does not export a fetch() function.");
    }
    return worker.fetch(request, env, ctx);
  }, "fetchDispatcher");
  return {
    ...worker,
    fetch(request, env, ctx) {
      const dispatcher = /* @__PURE__ */ __name(function(type, init) {
        if (type === "scheduled" && worker.scheduled !== void 0) {
          const controller = new __Facade_ScheduledController__(
            Date.now(),
            init.cron ?? "",
            () => {
            }
          );
          return worker.scheduled(controller, env, ctx);
        }
      }, "dispatcher");
      return __facade_invoke__(request, env, ctx, dispatcher, fetchDispatcher);
    }
  };
}
__name(wrapExportedHandler, "wrapExportedHandler");
function wrapWorkerEntrypoint(klass) {
  if (__INTERNAL_WRANGLER_MIDDLEWARE__ === void 0 || __INTERNAL_WRANGLER_MIDDLEWARE__.length === 0) {
    return klass;
  }
  for (const middleware of __INTERNAL_WRANGLER_MIDDLEWARE__) {
    __facade_register__(middleware);
  }
  return class extends klass {
    #fetchDispatcher = /* @__PURE__ */ __name((request, env, ctx) => {
      this.env = env;
      this.ctx = ctx;
      if (super.fetch === void 0) {
        throw new Error("Entrypoint class does not define a fetch() function.");
      }
      return super.fetch(request);
    }, "#fetchDispatcher");
    #dispatcher = /* @__PURE__ */ __name((type, init) => {
      if (type === "scheduled" && super.scheduled !== void 0) {
        const controller = new __Facade_ScheduledController__(
          Date.now(),
          init.cron ?? "",
          () => {
          }
        );
        return super.scheduled(controller);
      }
    }, "#dispatcher");
    fetch(request) {
      return __facade_invoke__(
        request,
        this.env,
        this.ctx,
        this.#dispatcher,
        this.#fetchDispatcher
      );
    }
  };
}
__name(wrapWorkerEntrypoint, "wrapWorkerEntrypoint");
var WRAPPED_ENTRY;
if (typeof middleware_insertion_facade_default === "object") {
  WRAPPED_ENTRY = wrapExportedHandler(middleware_insertion_facade_default);
} else if (typeof middleware_insertion_facade_default === "function") {
  WRAPPED_ENTRY = wrapWorkerEntrypoint(middleware_insertion_facade_default);
}
var middleware_loader_entry_default = WRAPPED_ENTRY;
export {
  __INTERNAL_WRANGLER_MIDDLEWARE__,
  middleware_loader_entry_default as default
};
//# sourceMappingURL=index.js.map
