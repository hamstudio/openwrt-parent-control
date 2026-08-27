// 家长控制系统前端交互逻辑 - 绿色健康守护版
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
    const tabs = ['members', 'devices', 'apps', 'settings'];
    tabs.forEach(t => {
        const el = document.getElementById('tab' + capitalize(t));
        const btn = document.getElementById('tabBtn' + capitalize(t));
        if (!el || !btn) return;

        if (t === tab) {
            el.classList.remove('hidden');
            btn.className = 'px-4 py-2 text-sm font-semibold rounded-xl bg-emerald-600 text-white shadow-sm transition whitespace-nowrap';
            if (t === 'apps') {
                btn.className += ' flex items-center space-x-1.5';
            }
        } else {
            el.classList.add('hidden');
            btn.className = 'px-4 py-2 text-sm font-semibold rounded-xl text-slate-600 dark:text-slate-300 hover:bg-emerald-50 dark:hover:bg-slate-800 transition whitespace-nowrap';
            if (t === 'apps') {
                btn.className += ' flex items-center space-x-1.5';
            }
        }
    });

    // 切换右上角主操作按钮
    const addMemberBtn = document.getElementById('addMemberBtn');
    const addAppBtn = document.getElementById('addAppBtn');
    if (tab === 'apps') {
        addMemberBtn.classList.add('hidden');
        addAppBtn.classList.remove('hidden');
        addAppBtn.classList.add('flex');
    } else {
        addMemberBtn.classList.remove('hidden');
        addAppBtn.classList.add('hidden');
        addAppBtn.classList.remove('flex');
    }

    if (tab === 'apps') {
        renderAppManagement();
    }

    lucide.createIcons();
}

function capitalize(s) {
    return s.charAt(0).toUpperCase() + s.slice(1);
}

// ======================= API 请求与状态获取 =======================
async function fetchStatus() {
    try {
        const res = await fetch('/api/status');
        const data = await res.json();
        appState.status = data;

        document.getElementById('statMembers').innerText = data.managed_members;
        document.getElementById('statDevices').innerText = `${data.active_devices} / ${data.total_devices}`;
        document.getElementById('statApps').innerText = data.app_count;
        document.getElementById('tabAppCountBadge').innerText = data.app_count;

        const badge = document.getElementById('kernelStatusBadge');
        if (data.kernel_dpi_ready) {
            badge.className = 'hidden md:flex items-center space-x-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-emerald-50 text-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-800';
            badge.innerHTML = '<span class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span><span>kmod-oaf DPI 引擎运行中</span>';
        } else {
            badge.className = 'hidden md:flex items-center space-x-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-amber-50 text-amber-700 dark:bg-amber-950/50 dark:text-amber-300 border border-amber-200 dark:border-amber-800';
            badge.innerHTML = '<span class="w-2 h-2 rounded-full bg-amber-500"></span><span>kmod-oaf 未加载 (降级模拟模式)</span>';
        }
    } catch (e) {
        console.error('Fetch status failed:', e);
    }
}

async function fetchCategories() {
    try {
        const res = await fetch('/api/apps');
        appState.categories = await res.json() || [];
        updateCategoryDropdowns();
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

// ======================= Tab 1: 渲染成员卡片 =======================
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
        let statusBadge = '<span class="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-emerald-100 text-emerald-800 dark:bg-emerald-950/60 dark:text-emerald-300">正常上网</span>';
        if (m.is_locked) {
            statusBadge = '<span class="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-rose-100 text-rose-700 dark:bg-rose-950/60 dark:text-rose-300 flex items-center space-x-1"><i data-lucide="lock" class="w-3 h-3"></i><span>已一键断网</span></span>';
        } else if (m.bonus_until && new Date(m.bonus_until) > new Date()) {
            statusBadge = '<span class="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-amber-100 text-amber-800 dark:bg-amber-950/60 dark:text-amber-300 flex items-center space-x-1"><i data-lucide="zap" class="w-3 h-3"></i><span>奖励加时中</span></span>';
        } else if (isQuotaExceeded) {
            statusBadge = '<span class="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-orange-100 text-orange-800 dark:bg-orange-950/60 dark:text-orange-300">配额已耗尽</span>';
        }

        // 多时间段计划概要
        let scheduleSummary = '<span class="text-slate-400">全天允许</span>';
        if (m.schedule && m.schedule.enabled && m.schedule.time_ranges && m.schedule.time_ranges.length > 0) {
            const isBlock = (m.schedule.action === 'block' || !m.schedule.action);
            const actionText = isBlock ? '🚫 禁网' : '✅ 仅允许';
            const actionColor = isBlock ? 'text-amber-600 dark:text-amber-400' : 'text-emerald-600 dark:text-emerald-400';
            
            const rangesText = m.schedule.time_ranges.map(tr => `${tr.start_time}~${tr.end_time}`).join(', ');
            
            let daysText = '每天';
            if (m.schedule.days && m.schedule.days.length > 0 && m.schedule.days.length < 7) {
                const dayMap = { 0: '周日', 1: '周一', 2: '周二', 3: '周三', 4: '周四', 5: '周五', 6: '周六' };
                if (JSON.stringify(m.schedule.days.slice().sort()) === JSON.stringify([1, 2, 3, 4, 5])) {
                    daysText = '工作日';
                } else if (JSON.stringify(m.schedule.days.slice().sort()) === JSON.stringify([0, 6])) {
                    daysText = '周末';
                } else {
                    daysText = m.schedule.days.map(d => dayMap[d]).join('');
                }
            }
            scheduleSummary = `<span class="${actionColor} font-semibold">${actionText}: ${rangesText} (${daysText})</span>`;
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
            <div class="bg-white dark:bg-slate-800 rounded-2xl p-5 border border-slate-200/80 dark:border-slate-700/80 shadow-sm flex flex-col justify-between space-y-4 hover:border-emerald-300 dark:hover:border-emerald-800/60 transition">
                <div class="flex items-start justify-between">
                    <div class="flex items-center space-x-3">
                        <div class="w-12 h-12 rounded-2xl bg-emerald-50 dark:bg-emerald-950/50 flex items-center justify-center avatar-badge border border-emerald-100 dark:border-emerald-900">
                            ${avatarEmoji}
                        </div>
                        <div>
                            <div class="flex items-center space-x-2">
                                <h3 class="font-bold text-base text-slate-800 dark:text-white">${m.name}</h3>
                                ${statusBadge}
                            </div>
                            <p class="text-xs text-slate-400 mt-0.5">绑定 ${m.device_macs ? m.device_macs.length : 0} 台设备 · 限制 ${m.blocked_app_ids ? m.blocked_app_ids.length : 0} 款 App</p>
                        </div>
                    </div>
                    <button onclick="editMember('${m.id}')" class="p-2 rounded-xl text-slate-400 hover:text-emerald-600 hover:bg-emerald-50 dark:hover:bg-slate-700 transition" title="编辑管控规则">
                        <i data-lucide="sliders" class="w-4 h-4"></i>
                    </button>
                </div>

                <!-- 时间段管控摘要 -->
                <div class="text-xs bg-slate-50 dark:bg-slate-900/50 p-2.5 rounded-xl border border-slate-100 dark:border-slate-700/60 flex items-center space-x-1.5">
                    <i data-lucide="clock" class="w-3.5 h-3.5 text-slate-400 flex-shrink-0"></i>
                    <div class="truncate">${scheduleSummary}</div>
                </div>

                <!-- 配额使用进度条 -->
                <div class="space-y-1.5 bg-slate-50 dark:bg-slate-900/40 p-3 rounded-xl border border-slate-100 dark:border-slate-700/50">
                    <div class="flex justify-between text-xs font-medium text-slate-500 dark:text-slate-400">
                        <span>今日已用活跃时长</span>
                        <span><b>${used}</b> / ${quota > 0 ? quota + ' 分钟' : '不限时'}</span>
                    </div>
                    <div class="w-full bg-slate-200 dark:bg-slate-700 h-2 rounded-full overflow-hidden">
                        <div class="h-full rounded-full transition-width ${percent > 90 ? 'bg-rose-500' : percent > 70 ? 'bg-amber-500' : 'bg-emerald-600'}" style="width: ${percent}%"></div>
                    </div>
                </div>

                <!-- 底部快捷控制按钮组 -->
                <div class="grid grid-cols-2 gap-2 pt-1">
                    ${m.is_locked ? `
                        <button onclick="unlockMember('${m.id}')" class="flex items-center justify-center space-x-1.5 py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-semibold shadow-sm transition">
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

// ======================= Tab 2: 局域网设备表格 =======================
function renderDevices() {
    const tbody = document.getElementById('devicesTableBody');
    if (!appState.devices || appState.devices.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" class="px-4 py-8 text-center text-slate-400">未发现局域网设备</td></tr>';
        return;
    }

    tbody.innerHTML = appState.devices.map(d => {
        let memberName = '<span class="text-slate-400 text-xs">未分配</span>';
        const member = appState.members.find(m => m.device_macs && m.device_macs.includes(d.mac));
        if (member) {
            memberName = `<span class="px-2 py-0.5 rounded-md bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400 font-medium text-xs">${member.name}</span>`;
        }

        const speedText = d.rx_rate > 1024 * 1024 
            ? (d.rx_rate / (1024 * 1024)).toFixed(1) + ' MB/s' 
            : (d.rx_rate / 1024).toFixed(0) + ' KB/s';

        return `
            <tr class="hover:bg-emerald-50/30 dark:hover:bg-slate-750/50 transition">
                <td class="px-4 py-3 font-semibold text-slate-800 dark:text-slate-100 flex items-center space-x-2">
                    <span class="w-2 h-2 rounded-full ${d.online ? 'bg-emerald-500' : 'bg-slate-300 dark:bg-slate-600'}"></span>
                    <span>${d.hostname || 'Unknown'}</span>
                </td>
                <td class="px-4 py-3 font-mono text-xs">${d.ip || '-'}</td>
                <td class="px-4 py-3 font-mono text-xs text-slate-500 dark:text-slate-400">${d.mac}</td>
                <td class="px-4 py-3 text-xs">${d.vendor || '通用设备'}</td>
                <td class="px-4 py-3 text-xs font-mono text-emerald-600 dark:text-emerald-400 font-semibold">${d.online ? speedText : '-'}</td>
                <td class="px-4 py-3">${memberName}</td>
                <td class="px-4 py-3 text-right">
                    <button onclick="quickAssignDevice('${d.mac}')" class="text-emerald-600 hover:text-emerald-700 dark:text-emerald-400 text-xs font-semibold">分配成员</button>
                </td>
            </tr>
        `;
    }).join('');
}

// ======================= Tab 3: 限制应用库管理 (CRUD 后台) =======================
function updateCategoryDropdowns() {
    const filterSelect = document.getElementById('appCategoryFilter');
    const formSelect = document.getElementById('formAppCategory');
    if (!filterSelect || !formSelect) return;

    const currentFilterVal = filterSelect.value;
    filterSelect.innerHTML = '<option value="all">全部分类</option>' + appState.categories.map(c => `
        <option value="${c.class_id}">${c.class_zh} (${c.apps ? c.apps.length : 0})</option>
    `).join('');
    filterSelect.value = currentFilterVal || 'all';

    formSelect.innerHTML = appState.categories.map(c => `
        <option value="${c.class_id}">${c.class_zh}</option>
    `).join('');
}

function renderAppManagement() {
    const container = document.getElementById('appManagementContainer');
    if (!container) return;

    const searchKey = (document.getElementById('appSearchInput')?.value || '').trim().toLowerCase();
    const catFilter = document.getElementById('appCategoryFilter')?.value || 'all';

    if (!appState.categories || appState.categories.length === 0) {
        container.innerHTML = '<div class="text-center py-12 text-slate-400">暂无应用分类特征数据</div>';
        return;
    }

    let filteredCats = appState.categories;
    if (catFilter !== 'all') {
        const filterId = parseInt(catFilter);
        filteredCats = filteredCats.filter(c => c.class_id === filterId);
    }

    let html = '';
    let totalMatchedApps = 0;

    filteredCats.forEach(cat => {
        const matchedApps = (cat.apps || []).filter(app => {
            if (!searchKey) return true;
            return app.name.toLowerCase().includes(searchKey) || String(app.id).includes(searchKey);
        });

        if (matchedApps.length === 0 && searchKey) return;
        totalMatchedApps += matchedApps.length;

        const appsRows = matchedApps.map(app => {
            return `
                <div class="flex items-center justify-between p-3 rounded-xl bg-slate-50 dark:bg-slate-900/50 border border-slate-200/70 dark:border-slate-700/60 hover:border-emerald-400 dark:hover:border-emerald-700 transition group">
                    <div class="flex items-center space-x-3">
                        <div class="w-8 h-8 rounded-lg bg-emerald-100 dark:bg-emerald-950/60 text-emerald-600 flex items-center justify-center font-bold text-xs">
                            ${app.name.charAt(0)}
                        </div>
                        <div>
                            <div class="flex items-center space-x-2">
                                <span class="font-bold text-sm text-slate-800 dark:text-slate-100">${app.name}</span>
                                <span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-slate-200 dark:bg-slate-700 text-slate-600 dark:text-slate-300">ID: ${app.id}</span>
                                ${app.is_custom ? '<span class="text-[10px] px-1.5 py-0.5 rounded-full bg-teal-100 text-teal-800 dark:bg-teal-950 dark:text-teal-300">自定义</span>' : ''}
                            </div>
                            <p class="text-xs text-slate-400 mt-0.5">${app.description || `${cat.class_zh} 类别 DPI 特征精准阻断`}</p>
                        </div>
                    </div>

                    <div class="flex items-center space-x-1.5">
                        <button onclick="editApp(${app.id})" class="p-1.5 rounded-lg text-slate-400 hover:text-emerald-600 hover:bg-emerald-50 dark:hover:bg-slate-800 transition" title="编辑应用">
                            <i data-lucide="edit-3" class="w-4 h-4"></i>
                        </button>
                        <button onclick="deleteApp(${app.id}, '${app.name}')" class="p-1.5 rounded-lg text-slate-400 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-slate-800 transition" title="删除应用">
                            <i data-lucide="trash-2" class="w-4 h-4"></i>
                        </button>
                    </div>
                </div>
            `;
        }).join('');

        html += `
            <div class="bg-white dark:bg-slate-800 rounded-2xl p-4 border border-slate-200 dark:border-slate-700 shadow-sm space-y-3">
                <div class="flex items-center justify-between pb-2 border-b border-slate-100 dark:border-slate-700">
                    <div class="flex items-center space-x-2">
                        <span class="font-bold text-base text-slate-800 dark:text-white flex items-center space-x-1.5">
                            <span>${cat.class_zh}</span>
                        </span>
                        <span class="text-xs px-2 py-0.5 rounded-full bg-emerald-50 text-emerald-600 dark:bg-emerald-950 dark:text-emerald-400 font-semibold">${matchedApps.length} 款 App</span>
                    </div>
                    <div class="flex items-center space-x-2">
                        <button onclick="openAppModalWithCategory(${cat.class_id})" class="text-xs font-semibold text-emerald-600 hover:text-emerald-700 flex items-center space-x-1">
                            <i data-lucide="plus" class="w-3.5 h-3.5"></i>
                            <span>添加此分类 App</span>
                        </button>
                        ${cat.is_custom ? `
                            <button onclick="deleteCategory(${cat.class_id}, '${cat.class_zh}')" class="text-xs text-rose-500 hover:text-rose-600 ml-2" title="删除分类">
                                <i data-lucide="trash" class="w-3.5 h-3.5"></i>
                            </button>
                        ` : ''}
                    </div>
                </div>

                <div class="grid grid-cols-1 md:grid-cols-2 gap-2.5">
                    ${appsRows || '<div class="text-xs text-slate-400 p-2 col-span-2">该分类下暂无匹配的应用</div>'}
                </div>
            </div>
        `;
    });

    if (totalMatchedApps === 0 && searchKey) {
        html = '<div class="text-center py-12 text-slate-400 bg-white dark:bg-slate-800 rounded-2xl border border-dashed border-slate-200 dark:border-slate-700">未检索到包含 "' + searchKey + '" 的受限应用</div>';
    }

    container.innerHTML = html;
    lucide.createIcons();
}

function filterAppList() {
    renderAppManagement();
}

// 限制应用 Modal 逻辑
function openAppModal(app = null) {
    const modal = document.getElementById('appModal');
    const title = document.getElementById('appModalTitle');
    updateCategoryDropdowns();

    if (app) {
        title.innerText = '编辑受限应用 - ' + app.name;
        document.getElementById('formAppId').value = app.id;
        document.getElementById('formAppName').value = app.name;
        document.getElementById('formAppCategory').value = app.class_id;
        document.getElementById('formAppDescription').value = app.description || '';
    } else {
        title.innerText = '新增受限应用';
        document.getElementById('formAppId').value = '';
        document.getElementById('formAppName').value = '';
        if (appState.categories.length > 0) {
            document.getElementById('formAppCategory').value = appState.categories[0].class_id;
        }
        document.getElementById('formAppDescription').value = '';
    }

    modal.classList.remove('hidden');
    lucide.createIcons();
}

function openAppModalWithCategory(classID) {
    openAppModal();
    document.getElementById('formAppCategory').value = classID;
}

function closeAppModal() {
    document.getElementById('appModal').classList.add('hidden');
}

function editApp(appID) {
    for (const cat of appState.categories) {
        const app = (cat.apps || []).find(a => a.id === appID);
        if (app) {
            openAppModal(app);
            return;
        }
    }
}

async function saveAppForm() {
    const idVal = document.getElementById('formAppId').value;
    const name = document.getElementById('formAppName').value.trim();
    const classId = parseInt(document.getElementById('formAppCategory').value);
    const description = document.getElementById('formAppDescription').value.trim();

    if (!name) {
        alert('请输入应用名称！');
        return;
    }

    const payload = {
        name: name,
        class_id: classId,
        description: description
    };

    try {
        let res;
        if (idVal) {
            payload.id = parseInt(idVal);
            res = await fetch(`/api/apps/${idVal}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
        } else {
            res = await fetch('/api/apps', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
        }

        if (res.ok) {
            closeAppModal();
            await fetchCategories();
            await fetchStatus();
            renderAppManagement();
            alert('应用已成功保存！');
        } else {
            const err = await res.json();
            alert('保存失败: ' + (err.error || '未知错误'));
        }
    } catch (e) {
        console.error('Save app failed:', e);
        alert('网络请求失败');
    }
}

async function deleteApp(appID, appName) {
    if (!confirm(`确定要删除应用 "${appName}" (ID: ${appID}) 吗？`)) return;

    try {
        const res = await fetch(`/api/apps/${appID}`, { method: 'DELETE' });
        if (res.ok) {
            await fetchCategories();
            await fetchMembers();
            await fetchStatus();
            renderAppManagement();
        } else {
            alert('删除失败');
        }
    } catch (e) {
        alert('网络请求错误');
    }
}

// 分类管理 Modal 逻辑
function openCategoryModal() {
    document.getElementById('categoryModal').classList.remove('hidden');
    document.getElementById('formCategoryName').value = '';
    lucide.createIcons();
}

function closeCategoryModal() {
    document.getElementById('categoryModal').classList.add('hidden');
}

async function saveCategoryForm() {
    const name = document.getElementById('formCategoryName').value.trim();
    const icon = document.getElementById('formCategoryIcon').value;
    if (!name) {
        alert('请输入分类名称！');
        return;
    }

    const payload = {
        class_zh: name,
        icon: icon
    };

    try {
        const res = await fetch('/api/categories', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        if (res.ok) {
            closeCategoryModal();
            await fetchCategories();
            renderAppManagement();
        } else {
            alert('新建分类失败');
        }
    } catch (e) {
        alert('网络请求失败');
    }
}

async function deleteCategory(classID, name) {
    if (!confirm(`确定要删除分类 "${name}" 吗？该分类下的应用将被归类为其他。`)) return;

    try {
        const res = await fetch(`/api/categories/${classID}`, { method: 'DELETE' });
        if (res.ok) {
            await fetchCategories();
            renderAppManagement();
        } else {
            alert('删除分类失败');
        }
    } catch (e) {
        alert('网络请求错误');
    }
}

// ======================= 成员管控与多时间段编辑器逻辑 =======================
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

        // 载入多时间段计划表
        const sched = member.schedule || {};
        document.getElementById('formScheduleEnabled').checked = sched.enabled !== false;
        
        // Action 单选
        const action = sched.action || 'block';
        const actionRadio = document.querySelector(`input[name="scheduleAction"][value="${action}"]`);
        if (actionRadio) actionRadio.checked = true;

        // 星期勾选
        const days = (sched.days && sched.days.length > 0) ? sched.days : [0, 1, 2, 3, 4, 5, 6];
        document.querySelectorAll('input[name="scheduleDay"]').forEach(cb => {
            cb.checked = days.includes(parseInt(cb.value));
        });

        // 渲染时间段列表
        const timeRanges = (sched.time_ranges && sched.time_ranges.length > 0) 
            ? sched.time_ranges 
            : [{ start_time: '21:30', end_time: '07:00' }];
        renderTimeRangeRows(timeRanges);

        btnDel.classList.remove('hidden');
    } else {
        title.innerText = '添加受管家庭成员';
        document.getElementById('formMemberId').value = '';
        document.getElementById('formMemberName').value = '';
        document.getElementById('formMemberAvatar').value = 'boy';
        document.getElementById('formQuotaMinutes').value = '120';
        document.getElementById('formScheduleEnabled').checked = true;
        
        const actionRadio = document.querySelector('input[name="scheduleAction"][value="block"]');
        if (actionRadio) actionRadio.checked = true;

        document.querySelectorAll('input[name="scheduleDay"]').forEach(cb => cb.checked = true);
        renderTimeRangeRows([{ start_time: '21:30', end_time: '07:00' }]);

        btnDel.classList.add('hidden');
    }

    toggleScheduleForm();
    renderModalDevices(member ? member.device_macs : []);
    renderModalAppCategories(member ? member.blocked_app_ids : []);

    modal.classList.remove('hidden');
    lucide.createIcons();
}

function closeMemberModal() {
    document.getElementById('memberModal').classList.add('hidden');
}

function toggleScheduleForm() {
    const isEnabled = document.getElementById('formScheduleEnabled').checked;
    const content = document.getElementById('scheduleFormContent');
    if (isEnabled) {
        content.classList.remove('opacity-40', 'pointer-events-none');
    } else {
        content.classList.add('opacity-40', 'pointer-events-none');
    }
}

function setDaysPreset(type) {
    const checkboxes = document.querySelectorAll('input[name="scheduleDay"]');
    checkboxes.forEach(cb => {
        const val = parseInt(cb.value);
        if (type === 'all') {
            cb.checked = true;
        } else if (type === 'workday') {
            cb.checked = (val >= 1 && val <= 5);
        } else if (type === 'weekend') {
            cb.checked = (val === 0 || val === 6);
        }
    });
}

function renderTimeRangeRows(ranges) {
    const container = document.getElementById('timeRangesList');
    container.innerHTML = ranges.map((r, idx) => `
        <div class="flex items-center space-x-2 bg-white dark:bg-slate-800 p-2 rounded-xl border border-slate-200 dark:border-slate-700 time-range-item">
            <span class="text-xs font-mono font-bold text-slate-400 w-5 text-center">${idx + 1}</span>
            <input type="time" value="${r.start_time || '21:30'}" class="range-start px-2.5 py-1.5 rounded-lg bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 text-xs text-slate-800 dark:text-slate-100">
            <span class="text-xs text-slate-400">至</span>
            <input type="time" value="${r.end_time || '07:00'}" class="range-end px-2.5 py-1.5 rounded-lg bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 text-xs text-slate-800 dark:text-slate-100">
            <button type="button" onclick="removeTimeRangeRow(this)" class="p-1.5 text-slate-400 hover:text-rose-600 rounded-lg ml-auto" title="删除该时段">
                <i data-lucide="trash-2" class="w-4 h-4"></i>
            </button>
        </div>
    `).join('');
    lucide.createIcons();
}

function addTimeRangeRow(startTime = '12:00', endTime = '14:00') {
    const container = document.getElementById('timeRangesList');
    const currentCount = container.querySelectorAll('.time-range-item').length;
    const row = document.createElement('div');
    row.className = 'flex items-center space-x-2 bg-white dark:bg-slate-800 p-2 rounded-xl border border-slate-200 dark:border-slate-700 time-range-item';
    row.innerHTML = `
        <span class="text-xs font-mono font-bold text-slate-400 w-5 text-center">${currentCount + 1}</span>
        <input type="time" value="${startTime}" class="range-start px-2.5 py-1.5 rounded-lg bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 text-xs text-slate-800 dark:text-slate-100">
        <span class="text-xs text-slate-400">至</span>
        <input type="time" value="${endTime}" class="range-end px-2.5 py-1.5 rounded-lg bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 text-xs text-slate-800 dark:text-slate-100">
        <button type="button" onclick="removeTimeRangeRow(this)" class="p-1.5 text-slate-400 hover:text-rose-600 rounded-lg ml-auto" title="删除该时段">
            <i data-lucide="trash-2" class="w-4 h-4"></i>
        </button>
    `;
    container.appendChild(row);
    lucide.createIcons();
}

function removeTimeRangeRow(btn) {
    const item = btn.closest('.time-range-item');
    if (item) {
        item.remove();
        // 重新编号
        document.querySelectorAll('#timeRangesList .time-range-item').forEach((row, i) => {
            row.querySelector('span').innerText = i + 1;
        });
    }
}

function applyPresetSchedule(preset) {
    if (preset === 'night') {
        renderTimeRangeRows([{ start_time: '21:30', end_time: '07:00' }]);
    } else if (preset === 'school') {
        renderTimeRangeRows([
            { start_time: '08:00', end_time: '11:30' },
            { start_time: '14:00', end_time: '17:30' },
            { start_time: '21:30', end_time: '07:00' }
        ]);
    }
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
            <label class="flex items-center justify-between p-2 rounded-lg bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 cursor-pointer hover:border-emerald-500 transition">
                <div class="flex items-center space-x-2">
                    <input type="checkbox" name="modalDevice" value="${d.mac}" ${isChecked ? 'checked' : ''} class="w-4 h-4 text-emerald-600 rounded">
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
        const appsHTML = (cat.apps || []).map(app => {
            const isChecked = selectedSet.has(app.id);
            return `
                <label class="inline-flex items-center space-x-1.5 bg-white dark:bg-slate-800 px-2.5 py-1.5 rounded-lg border border-slate-200 dark:border-slate-700 cursor-pointer text-xs hover:border-emerald-500">
                    <input type="checkbox" name="modalApp" value="${app.id}" ${isChecked ? 'checked' : ''} onchange="updateSelectedCount()" class="w-3.5 h-3.5 text-emerald-600 rounded">
                    <span>${app.name}</span>
                </label>
            `;
        }).join('');

        return `
            <div class="border border-slate-200 dark:border-slate-700 rounded-xl p-2.5 bg-slate-50/50 dark:bg-slate-900/30">
                <div class="flex items-center justify-between mb-2">
                    <span class="font-bold text-xs text-slate-700 dark:text-slate-200 flex items-center space-x-1">
                        <span>${cat.class_zh}</span>
                        <span class="text-[11px] text-slate-400">(${(cat.apps || []).length})</span>
                    </span>
                    <button type="button" onclick="toggleSelectAllCategory(${cat.class_id})" class="text-[11px] text-emerald-600 hover:underline">全选/反选</button>
                </div>
                <div class="flex flex-wrap gap-1.5" data-cat-id="${cat.class_id}">
                    ${appsHTML || '<span class="text-xs text-slate-400">暂无 App</span>'}
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

// 保存成员表单 (包含多时间段与动作模式)
async function saveMemberForm() {
    const id = document.getElementById('formMemberId').value || 'm_' + Date.now();
    const name = document.getElementById('formMemberName').value.trim();
    if (!name) {
        alert('请输入成员姓名！');
        return;
    }

    const avatar = document.getElementById('formMemberAvatar').value;
    const quota = parseInt(document.getElementById('formQuotaMinutes').value) || 0;

    // 解析时间段列表
    const scheduleEnabled = document.getElementById('formScheduleEnabled').checked;
    const actionVal = document.querySelector('input[name="scheduleAction"]:checked')?.value || 'block';
    
    const selectedDays = Array.from(document.querySelectorAll('input[name="scheduleDay"]:checked')).map(cb => parseInt(cb.value));
    
    const timeRangeRows = document.querySelectorAll('#timeRangesList .time-range-item');
    const timeRanges = [];
    timeRangeRows.forEach(row => {
        const s = row.querySelector('.range-start')?.value;
        const e = row.querySelector('.range-end')?.value;
        if (s && e) {
            timeRanges.push({ start_time: s, end_time: e });
        }
    });

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
            enabled: scheduleEnabled && timeRanges.length > 0,
            days: selectedDays.length > 0 ? selectedDays : [0, 1, 2, 3, 4, 5, 6],
            time_ranges: timeRanges,
            action: actionVal
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
            await fetchStatus();
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
        await fetchStatus();
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
            alert('全局健康守护设置已成功保存并立即生效！');
        }
    } catch (e) {
        alert('保存设置失败');
    }
}

