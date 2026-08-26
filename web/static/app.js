// 家长控制系统前端交互逻辑
let appState = {
    members: [],
    devices: [],
    categories: [],
    settings: {},
    status: {}
};

// 初始化
document.addEventListener('DOMContentLoaded', async () => {
    initTheme();
    lucide.createIcons();
    await Promise.all([
        fetchStatus(),
        fetchCategories(),
        fetchMembers(),
        fetchDevices(),
        fetchSettings()
    ]);
    lucide.createIcons();

    // 5秒轮询刷新状态与设备
    setInterval(async () => {
        await Promise.all([fetchStatus(), fetchMembers(), fetchDevices()]);
        lucide.createIcons();
    }, 5000);
});

// 主题切换
function initTheme() {
    const isDark = localStorage.theme === 'dark' || (!('theme' in localStorage) && window.matchMedia('(prefers-color-scheme: dark)').matches);
    if (isDark) {
        document.documentElement.classList.add('dark');
    } else {
        document.documentElement.classList.remove('dark');
    }

    document.getElementById('themeToggle').addEventListener('click', () => {
        document.documentElement.classList.toggle('dark');
        localStorage.theme = document.documentElement.classList.contains('dark') ? 'dark' : 'light';
    });
}

// 标签页切换
function switchTab(tab) {
    const tabs = ['members', 'devices', 'settings'];
    tabs.forEach(t => {
        const el = document.getElementById('tab' + capitalize(t));
        const btn = document.getElementById('tabBtn' + capitalize(t));
        if (t === tab) {
            el.classList.remove('hidden');
            btn.className = 'px-4 py-2 text-sm font-semibold rounded-xl bg-indigo-600 text-white shadow-sm transition';
        } else {
            el.classList.add('hidden');
            btn.className = 'px-4 py-2 text-sm font-semibold rounded-xl text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 transition';
        }
    });
    lucide.createIcons();
}

function capitalize(s) {
    return s.charAt(0).toUpperCase() + s.slice(1);
}

// API 请求与渲染
async function fetchStatus() {
    try {
        const res = await fetch('/api/status');
        const data = await res.json();
        appState.status = data;

        document.getElementById('statMembers').innerText = data.managed_members;
        document.getElementById('statDevices').innerText = `${data.active_devices} / ${data.total_devices}`;
        document.getElementById('statApps').innerText = data.app_count;

        const badge = document.getElementById('kernelStatusBadge');
        if (data.kernel_dpi_ready) {
            badge.className = 'hidden md:flex items-center space-x-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-emerald-50 text-emerald-600 dark:bg-emerald-950/50 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800';
            badge.innerHTML = '<span class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span><span>kmod-oaf DPI 引擎运行中</span>';
        } else {
            badge.className = 'hidden md:flex items-center space-x-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-amber-50 text-amber-600 dark:bg-amber-950/50 dark:text-amber-400 border border-amber-200 dark:border-amber-800';
            badge.innerHTML = '<span class="w-2 h-2 rounded-full bg-amber-500"></span><span>kmod-oaf 未加载 (规则降级)</span>';
        }
    } catch (e) {
        console.error('Fetch status failed:', e);
    }
}

async function fetchCategories() {
    try {
        const res = await fetch('/api/apps');
        appState.categories = await res.json();
    } catch (e) {
        console.error('Fetch apps failed:', e);
    }
}

async function fetchMembers() {
    try {
        const res = await fetch('/api/members');
        appState.members = await res.json() || [];
        renderMembers();
    } catch (e) {
        console.error('Fetch members failed:', e);
    }
}

async function fetchDevices() {
    try {
        const res = await fetch('/api/devices');
        appState.devices = await res.json() || [];
        renderDevices();
    } catch (e) {
        console.error('Fetch devices failed:', e);
    }
}

async function fetchSettings() {
    try {
        const res = await fetch('/api/settings');
        appState.settings = await res.json();
        document.getElementById('settingGlobalEnable').checked = appState.settings.enabled;
        document.getElementById('settingSafeSearch').checked = appState.settings.enforce_safe_search;
        document.getElementById('settingBlockDoH').checked = appState.settings.block_doh_dot;
        document.getElementById('settingIsolateNew').checked = appState.settings.isolate_new_devices;
    } catch (e) {
        console.error('Fetch settings failed:', e);
    }
}

// 渲染成员卡片
function renderMembers() {
    const container = document.getElementById('membersContainer');
    const emptyEl = document.getElementById('emptyMembers');

    if (!appState.members || appState.members.length === 0) {
        container.innerHTML = '';
        emptyEl.classList.remove('hidden');
        return;
    }
    emptyEl.classList.add('hidden');

    container.innerHTML = appState.members.map(m => {
        // 计算配额进度
        const quota = m.quota_minutes || 0;
        const used = m.used_minutes || 0;
        const percent = quota > 0 ? Math.min(100, Math.round((used / quota) * 100)) : 0;
        const isQuotaExceeded = quota > 0 && used >= quota;

        // 状态徽章
        let statusBadge = '<span class="px-2 py-0.5 rounded-full text-xs font-semibold bg-emerald-100 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400">正常上网</span>';
        if (m.is_locked) {
            statusBadge = '<span class="px-2 py-0.5 rounded-full text-xs font-semibold bg-red-100 text-red-700 dark:bg-red-950/60 dark:text-red-400 flex items-center space-x-1"><i data-lucide="lock" class="w-3 h-3"></i><span>已一键断网</span></span>';
        } else if (m.bonus_until && new Date(m.bonus_until) > new Date()) {
            statusBadge = '<span class="px-2 py-0.5 rounded-full text-xs font-semibold bg-amber-100 text-amber-700 dark:bg-amber-950/60 dark:text-amber-400 flex items-center space-x-1"><i data-lucide="zap" class="w-3 h-3"></i><span>奖励加时中</span></span>';
        } else if (isQuotaExceeded) {
            statusBadge = '<span class="px-2 py-0.5 rounded-full text-xs font-semibold bg-orange-100 text-orange-700 dark:bg-orange-950/60 dark:text-orange-400">配额已耗尽</span>';
        }

        // 头像映射
        const avatarMap = {
            boy: '👦',
            girl: '👧',
            student: '🧑‍🎓',
            child: '👶'
        };
        const avatarEmoji = avatarMap[m.avatar] || '👦';

        return `
            <div class="bg-white dark:bg-slate-800 rounded-2xl p-5 border border-slate-200/80 dark:border-slate-700/80 shadow-sm flex flex-col justify-between space-y-4">
                <div class="flex items-start justify-between">
                    <div class="flex items-center space-x-3">
                        <div class="w-12 h-12 rounded-2xl bg-indigo-50 dark:bg-indigo-950/50 flex items-center justify-center avatar-badge border border-indigo-100 dark:border-indigo-900">
                            ${avatarEmoji}
                        </div>
                        <div>
                            <div class="flex items-center space-x-2">
                                <h3 class="font-bold text-base text-slate-800 dark:text-white">${m.name}</h3>
                                ${statusBadge}
                            </div>
                            <p class="text-xs text-slate-400 mt-0.5">绑定 ${m.device_macs ? m.device_macs.length : 0} 台设备 · 封禁 ${m.blocked_app_ids ? m.blocked_app_ids.length : 0} 款 App</p>
                        </div>
                    </div>
                    <button onclick="editMember('${m.id}')" class="p-2 rounded-xl text-slate-400 hover:text-indigo-600 hover:bg-indigo-50 dark:hover:bg-slate-700 transition">
                        <i data-lucide="sliders" class="w-4 h-4"></i>
                    </button>
                </div>

                <!-- 配额使用进度条 -->
                <div class="space-y-1.5 bg-slate-50 dark:bg-slate-900/40 p-3 rounded-xl border border-slate-100 dark:border-slate-700/50">
                    <div class="flex justify-between text-xs font-medium text-slate-500 dark:text-slate-400">
                        <span>今日已用活跃时长</span>
                        <span><b>${used}</b> / ${quota > 0 ? quota + ' 分钟' : '不限时'}</span>
                    </div>
                    <div class="w-full bg-slate-200 dark:bg-slate-700 h-2 rounded-full overflow-hidden">
                        <div class="h-full rounded-full transition-width ${percent > 90 ? 'bg-red-500' : percent > 70 ? 'bg-amber-500' : 'bg-indigo-600'}" style="width: ${percent}%"></div>
                    </div>
                </div>

                <!-- 底部快捷控制按钮组 -->
                <div class="grid grid-cols-2 gap-2 pt-1">
                    ${m.is_locked ? `
                        <button onclick="unlockMember('${m.id}')" class="flex items-center justify-center space-x-1.5 py-2.5 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-white text-xs font-semibold shadow-sm transition">
                            <i data-lucide="unlock" class="w-3.5 h-3.5"></i>
                            <span>恢复上网</span>
                        </button>
                    ` : `
                        <button onclick="lockMember('${m.id}')" class="flex items-center justify-center space-x-1.5 py-2.5 rounded-xl bg-rose-500 hover:bg-rose-600 text-white text-xs font-semibold shadow-sm transition">
                            <i data-lucide="lock" class="w-3.5 h-3.5"></i>
                            <span>一键断网</span>
                        </button>
                    `}
                    <button onclick="bonusMember('${m.id}', 30)" class="flex items-center justify-center space-x-1.5 py-2.5 rounded-xl bg-amber-500 hover:bg-amber-600 text-white text-xs font-semibold shadow-sm transition">
                        <i data-lucide="plus-circle" class="w-3.5 h-3.5"></i>
                        <span>奖励 +30分钟</span>
                    </button>
                </div>
            </div>
        `;
    }).join('');
}

// 渲染局域网设备表格
function renderDevices() {
    const tbody = document.getElementById('devicesTableBody');
    if (!appState.devices || appState.devices.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" class="px-4 py-8 text-center text-slate-400">未发现局域网设备</td></tr>';
        return;
    }

    tbody.innerHTML = appState.devices.map(d => {
        // 查找归属成员
        let memberName = '<span class="text-slate-400 text-xs">未分配</span>';
        const member = appState.members.find(m => m.device_macs && m.device_macs.includes(d.mac));
        if (member) {
            memberName = `<span class="px-2 py-0.5 rounded-md bg-indigo-50 text-indigo-600 dark:bg-indigo-950/60 dark:text-indigo-400 font-medium text-xs">${member.name}</span>`;
        }

        const speedText = d.rx_rate > 1024 * 1024 
            ? (d.rx_rate / (1024 * 1024)).toFixed(1) + ' MB/s' 
            : (d.rx_rate / 1024).toFixed(0) + ' KB/s';

        return `
            <tr class="hover:bg-slate-50/50 dark:hover:bg-slate-750/50 transition">
                <td class="px-4 py-3 font-semibold text-slate-800 dark:text-slate-100 flex items-center space-x-2">
                    <span class="w-2 h-2 rounded-full ${d.online ? 'bg-emerald-500' : 'bg-slate-300 dark:bg-slate-600'}"></span>
                    <span>${d.hostname || 'Unknown'}</span>
                </td>
                <td class="px-4 py-3 font-mono text-xs">${d.ip || '-'}</td>
                <td class="px-4 py-3 font-mono text-xs text-slate-500 dark:text-slate-400">${d.mac}</td>
                <td class="px-4 py-3 text-xs">${d.vendor || '通用设备'}</td>
                <td class="px-4 py-3 text-xs font-mono text-indigo-600 dark:text-indigo-400 font-semibold">${d.online ? speedText : '-'}</td>
                <td class="px-4 py-3">${memberName}</td>
                <td class="px-4 py-3 text-right">
                    <button onclick="quickAssignDevice('${d.mac}')" class="text-indigo-600 hover:text-indigo-700 text-xs font-semibold">分配成员</button>
                </td>
            </tr>
        `;
    }).join('');
}

// 成员编辑与创建 Modal 逻辑
function openMemberModal(member = null) {
    const modal = document.getElementById('memberModal');
    const title = document.getElementById('modalTitle');
    const btnDel = document.getElementById('btnDeleteMember');

    if (member) {
        title.innerText = '编辑成员规则 - ' + member.name;
        document.getElementById('formMemberId').value = member.id;
        document.getElementById('formMemberName').value = member.name;
        document.getElementById('formMemberAvatar').value = member.avatar || 'boy';
        document.getElementById('formQuotaMinutes').value = member.quota_minutes || '';
        if (member.schedule && member.schedule.time_ranges && member.schedule.time_ranges.length > 0) {
            document.getElementById('formStartTime').value = member.schedule.time_ranges[0].start_time;
            document.getElementById('formEndTime').value = member.schedule.time_ranges[0].end_time;
        }
        btnDel.classList.remove('hidden');
    } else {
        title.innerText = '添加受管家庭成员';
        document.getElementById('formMemberId').value = '';
        document.getElementById('formMemberName').value = '';
        document.getElementById('formMemberAvatar').value = 'boy';
        document.getElementById('formQuotaMinutes').value = '120';
        document.getElementById('formStartTime').value = '21:30';
        document.getElementById('formEndTime').value = '07:00';
        btnDel.classList.add('hidden');
    }

    renderModalDevices(member ? member.device_macs : []);
    renderModalAppCategories(member ? member.blocked_app_ids : []);

    modal.classList.remove('hidden');
    lucide.createIcons();
}

function closeMemberModal() {
    document.getElementById('memberModal').classList.add('hidden');
}

function renderModalDevices(selectedMACs = []) {
    const container = document.getElementById('modalDeviceList');
    if (!appState.devices || appState.devices.length === 0) {
        container.innerHTML = '<div class="text-xs text-slate-400 p-2">未探测到局域网设备</div>';
        return;
    }

    container.innerHTML = appState.devices.map(d => {
        const isChecked = selectedMACs && selectedMACs.includes(d.mac);
        return `
            <label class="flex items-center justify-between p-2 rounded-lg bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 cursor-pointer hover:border-indigo-500 transition">
                <div class="flex items-center space-x-2">
                    <input type="checkbox" name="modalDevice" value="${d.mac}" ${isChecked ? 'checked' : ''} class="w-4 h-4 text-indigo-600 rounded">
                    <span class="font-medium text-xs text-slate-800 dark:text-slate-100">${d.hostname}</span>
                    <span class="text-[11px] text-slate-400 font-mono">(${d.ip} / ${d.vendor})</span>
                </div>
            </label>
        `;
    }).join('');
}

function renderModalAppCategories(selectedAppIDs = []) {
    const container = document.getElementById('modalAppCategories');
    const selectedSet = new Set(selectedAppIDs || []);

    if (!appState.categories || appState.categories.length === 0) {
        container.innerHTML = '<div class="text-xs text-slate-400 p-2">加载特征库中...</div>';
        return;
    }

    container.innerHTML = appState.categories.map(cat => {
        const appsHTML = cat.apps.map(app => {
            const isChecked = selectedSet.has(app.id);
            return `
                <label class="inline-flex items-center space-x-1.5 bg-white dark:bg-slate-800 px-2.5 py-1.5 rounded-lg border border-slate-200 dark:border-slate-700 cursor-pointer text-xs">
                    <input type="checkbox" name="modalApp" value="${app.id}" ${isChecked ? 'checked' : ''} onchange="updateSelectedCount()" class="w-3.5 h-3.5 text-indigo-600 rounded">
                    <span>${app.name}</span>
                </label>
            `;
        }).join('');

        return `
            <div class="border border-slate-200 dark:border-slate-700 rounded-xl p-2.5 bg-slate-50/50 dark:bg-slate-900/30">
                <div class="flex items-center justify-between mb-2">
                    <span class="font-bold text-xs text-slate-700 dark:text-slate-200 flex items-center space-x-1">
                        <span>${cat.class_zh}</span>
                    </span>
                    <button type="button" onclick="toggleSelectAllCategory(${cat.class_id})" class="text-[11px] text-indigo-600 hover:underline">全选/反选</button>
                </div>
                <div class="flex flex-wrap gap-1.5" data-cat-id="${cat.class_id}">
                    ${appsHTML}
                </div>
            </div>
        `;
    }).join('');

    updateSelectedCount();
}

function updateSelectedCount() {
    const checked = document.querySelectorAll('input[name="modalApp"]:checked');
    document.getElementById('selectedAppCount').innerText = checked.length;
}

function toggleSelectAllCategory(classID) {
    const container = document.querySelector(`div[data-cat-id="${classID}"]`);
    if (!container) return;
    const checkboxes = container.querySelectorAll('input[name="modalApp"]');
    const allChecked = Array.from(checkboxes).every(cb => cb.checked);
    checkboxes.forEach(cb => cb.checked = !allChecked);
    updateSelectedCount();
}

// 快速分配设备
function quickAssignDevice(mac) {
    openMemberModal();
    const cb = document.querySelector(`input[name="modalDevice"][value="${mac}"]`);
    if (cb) cb.checked = true;
}

// 编辑成员
function editMember(id) {
    const member = appState.members.find(m => m.id === id);
    if (member) {
        openMemberModal(member);
    }
}

// 保存成员表单
async function saveMemberForm() {
    const id = document.getElementById('formMemberId').value || 'm_' + Date.now();
    const name = document.getElementById('formMemberName').value.trim();
    if (!name) {
        alert('请输入成员姓名！');
        return;
    }

    const avatar = document.getElementById('formMemberAvatar').value;
    const quota = parseInt(document.getElementById('formQuotaMinutes').value) || 0;
    const startTime = document.getElementById('formStartTime').value;
    const endTime = document.getElementById('formEndTime').value;

    const deviceMACs = Array.from(document.querySelectorAll('input[name="modalDevice"]:checked')).map(cb => cb.value);
    const blockedAppIDs = Array.from(document.querySelectorAll('input[name="modalApp"]:checked')).map(cb => parseInt(cb.value));

    const payload = {
        id: id,
        name: name,
        avatar: avatar,
        device_macs: deviceMACs,
        enabled: true,
        quota_minutes: quota,
        schedule: {
            enabled: startTime !== '' && endTime !== '',
            days: [0, 1, 2, 3, 4, 5, 6],
            time_ranges: [{ start_time: startTime, end_time: endTime }],
            action: 'block'
        },
        blocked_app_ids: blockedAppIDs,
        safe_search: true,
        block_adult: true
    };

    try {
        const res = await fetch('/api/members', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        if (res.ok) {
            closeMemberModal();
            await fetchMembers();
            lucide.createIcons();
        } else {
            alert('保存失败，请检查参数');
        }
    } catch (e) {
        console.error('Save member failed:', e);
        alert('网络请求失败');
    }
}

// 删除成员
async function deleteCurrentMember() {
    const id = document.getElementById('formMemberId').value;
    if (!id || !confirm('确定要删除该受管成员吗？')) return;

    try {
        await fetch(`/api/members/${id}`, { method: 'DELETE' });
        closeMemberModal();
        await fetchMembers();
        lucide.createIcons();
    } catch (e) {
        alert('删除失败');
    }
}

// 一键断网
async function lockMember(id) {
    try {
        await fetch(`/api/members/${id}/lock`, { method: 'POST' });
        await fetchMembers();
        lucide.createIcons();
    } catch (e) {
        alert('操作失败');
    }
}

// 恢复上网
async function unlockMember(id) {
    try {
        await fetch(`/api/members/${id}/unlock`, { method: 'POST' });
        await fetchMembers();
        lucide.createIcons();
    } catch (e) {
        alert('操作失败');
    }
}

// 奖励加时
async function bonusMember(id, minutes) {
    try {
        await fetch(`/api/members/${id}/bonus?minutes=${minutes}`, { method: 'POST' });
        await fetchMembers();
        lucide.createIcons();
    } catch (e) {
        alert('操作失败');
    }
}

// 保存全局设置
async function saveGlobalSettings() {
    const payload = {
        enabled: document.getElementById('settingGlobalEnable').checked,
        enforce_safe_search: document.getElementById('settingSafeSearch').checked,
        block_doh_dot: document.getElementById('settingBlockDoH').checked,
        isolate_new_devices: document.getElementById('settingIsolateNew').checked,
    };

    try {
        const res = await fetch('/api/settings', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        if (res.ok) {
            alert('全局设置已成功保存并立即生效！');
        }
    } catch (e) {
        alert('保存设置失败');
    }
}
