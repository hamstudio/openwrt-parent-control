// 家长控制系统前端交互逻辑 - 完整全功能版
// 安全兜底保护
if (typeof window.t !== 'function') window.t = (k) => k;
if (typeof window.getLocale !== 'function') window.getLocale = () => 'zh-CN';
if (typeof window.tDpiCategory !== 'function') window.tDpiCategory = (name) => name;
if (typeof window.tDpiApp !== 'function') window.tDpiApp = (name) => name;

let appState = {
    members: [],
    devices: [],
    categories: [],
    settings: {},
    status: {},
    currentPinInput: '',
    editingAppCatId: null
};

// 获取已存储的 PIN 码
function getStoredPin() {
    return localStorage.getItem('parentcontrol_pin') || '';
}

function setStoredPin(pin) {
    if (pin) {
        localStorage.setItem('parentcontrol_pin', pin);
    } else {
        localStorage.removeItem('parentcontrol_pin');
    }
}

// 封装带 PIN 鉴权的 Fetch
async function authFetch(url, options = {}) {
    options.headers = options.headers || {};
    const pin = getStoredPin();
    if (pin) {
        if (options.headers instanceof Headers) {
            options.headers.set('X-Pin-Code', pin);
        } else {
            options.headers['X-Pin-Code'] = pin;
        }
    }

    const res = await fetch(url, options);
    if (res.status === 401) {
        openPinLockModal();
        throw new Error('Unauthorized: PIN code required');
    }
    return res;
}

// PIN 码输入键盘逻辑
function openPinLockModal() {
    appState.currentPinInput = '';
    updatePinDots();
    document.getElementById('pinLockModal').classList.remove('hidden');
    lucide.createIcons();
}

function closePinLockModal() {
    document.getElementById('pinLockModal').classList.add('hidden');
    appState.currentPinInput = '';
}

function lockConsole() {
    setStoredPin('');
    openPinLockModal();
}

function pressPinKey(digit) {
    if (appState.currentPinInput.length < 4) {
        appState.currentPinInput += digit;
        updatePinDots();
        if (appState.currentPinInput.length === 4) {
            submitPinVerification();
        }
    }
}

function deletePinKey() {
    if (appState.currentPinInput.length > 0) {
        appState.currentPinInput = appState.currentPinInput.slice(0, -1);
        updatePinDots();
    }
}

function clearPinKey() {
    appState.currentPinInput = '';
    updatePinDots();
}

function updatePinDots() {
    const len = appState.currentPinInput.length;
    for (let i = 0; i < 4; i++) {
        const dot = document.getElementById('pinDot' + i);
        if (i < len) {
            dot.className = 'w-3.5 h-3.5 rounded-full bg-emerald-600 border-2 border-emerald-600 scale-110 transition-all';
        } else {
            dot.className = 'w-3.5 h-3.5 rounded-full border-2 border-slate-300 dark:border-slate-600 transition-all';
        }
    }
}

async function submitPinVerification() {
    const enteredPin = appState.currentPinInput;
    try {
        const res = await fetch('/api/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ pin: enteredPin })
        });
        const data = await res.json();
        if (res.ok && data.success) {
            setStoredPin(enteredPin);
            closePinLockModal();
            showToast(t('toastAuthSuccess'), 'success');
            await Promise.all([fetchStatus(), fetchMembers(), fetchDevices(), fetchCategories(), fetchSettings()]);
            lucide.createIcons();
        } else {
            showToast(t('toastPinError'), 'error');
            clearPinKey();
        }
    } catch (e) {
        showToast(t('toastAuthFailed'), 'error');
        clearPinKey();
    }
}

// 物理键盘监听
document.addEventListener('keydown', (e) => {
    const modal = document.getElementById('pinLockModal');
    if (!modal.classList.contains('hidden')) {
        if (e.key >= '0' && e.key <= '9') {
            pressPinKey(e.key);
        } else if (e.key === 'Backspace') {
            deletePinKey();
        } else if (e.key === 'Escape') {
            clearPinKey();
        }
    }
});

// Toast 提示通知系统
function showToast(message, type = 'success') {
    const container = document.getElementById('toastContainer');
    if (!container) return;

    const toast = document.createElement('div');
    toast.className = 'toast-enter flex items-center space-x-2 px-4 py-2.5 rounded-xl shadow-lg text-xs font-semibold backdrop-blur-md transition-all duration-300 pointer-events-auto';

    let icon = 'check-circle';
    if (type === 'success') {
        toast.classList.add('bg-emerald-500/90', 'text-white', 'shadow-emerald-500/20');
        icon = 'check-circle';
    } else if (type === 'error') {
        toast.classList.add('bg-rose-500/90', 'text-white', 'shadow-rose-500/20');
        icon = 'alert-circle';
    } else if (type === 'warning') {
        toast.classList.add('bg-amber-500/90', 'text-white', 'shadow-amber-500/20');
        icon = 'alert-triangle';
    } else {
        toast.classList.add('bg-slate-800/90', 'text-white', 'dark:bg-slate-700/90');
        icon = 'info';
    }

    toast.innerHTML = `<i data-lucide="${icon}" class="w-4 h-4"></i><span>${message}</span>`;
    container.appendChild(toast);
    lucide.createIcons();

    requestAnimationFrame(() => {
        toast.classList.remove('toast-enter');
        toast.classList.add('toast-active');
    });

    setTimeout(() => {
        toast.classList.remove('toast-active');
        toast.classList.add('toast-exit');
        setTimeout(() => {
            toast.remove();
        }, 260);
    }, 2800);
}

// 国际化联动
function initI18n() {
    currentLocale = getLocale();
    const select = document.getElementById('langSelect');
    if (select) {
        select.value = currentLocale;
    }
    applyI18n();
}

function changeLanguage(lang) {
    setLocale(lang);
}

function applyI18n() {
    // 1. 更新所有 data-i18n 属性的静态 DOM 节点
    document.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.getAttribute('data-i18n');
        if (key) {
            el.innerText = t(key);
        }
    });

    // 2. 更新所有 data-i18n-placeholder 属性的输入框
    document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
        const key = el.getAttribute('data-i18n-placeholder');
        if (key) {
            el.setAttribute('placeholder', t(key));
        }
    });

    // 3. 更新 Tab 按钮文案
    const btnMembers = document.getElementById('tabBtnMembers');
    if (btnMembers) btnMembers.innerText = t('tabMembers');

    const btnDevices = document.getElementById('tabBtnDevices');
    if (btnDevices) btnDevices.innerText = t('tabDevices');

    const btnSettings = document.getElementById('tabBtnSettings');
    if (btnSettings) btnSettings.innerText = t('tabSettings');

    // 4. 重新渲染动态列表
    renderMembers();
    renderDevices();
    if (!document.getElementById('tabApps').classList.contains('hidden')) {
        renderAppManagement();
    }
    lucide.createIcons();
}

// 初始化
document.addEventListener('DOMContentLoaded', async () => {
    initTheme();
    initI18n();
    lucide.createIcons();
    await fetchStatus();

    if (appState.status && appState.status.pin_required && !getStoredPin()) {
        openPinLockModal();
    } else {
        await Promise.all([
            fetchCategories(),
            fetchMembers(),
            fetchDevices(),
            fetchSettings()
        ]);
    }
    lucide.createIcons();

    // 5秒轮询
    setInterval(async () => {
        if (document.getElementById('pinLockModal').classList.contains('hidden')) {
            await Promise.all([fetchStatus(), fetchMembers(), fetchDevices()]);
            lucide.createIcons();
        }
    }, 5000);
});

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

    const addMemberBtn = document.getElementById('addMemberBtn');
    const addAppBtn = document.getElementById('addAppBtn');
    if (tab === 'apps') {
        if (addMemberBtn) addMemberBtn.classList.add('hidden');
        if (addAppBtn) {
            addAppBtn.classList.remove('hidden');
            addAppBtn.classList.add('flex');
        }
        renderAppManagement();
    } else {
        if (addMemberBtn) addMemberBtn.classList.remove('hidden');
        if (addAppBtn) {
            addAppBtn.classList.add('hidden');
            addAppBtn.classList.remove('flex');
        }
    }

    lucide.createIcons();
}

function capitalize(s) {
    return s.charAt(0).toUpperCase() + s.slice(1);
}

// API 数据拉取
async function fetchStatus() {
    const start = performance.now();
    try {
        const res = await fetch('/api/status');
        const data = await res.json();
        const latency = Math.round(performance.now() - start);
        appState.status = data;

        document.getElementById('statMembers').innerText = data.managed_members;
        document.getElementById('statDevices').innerText = `${data.active_devices} / ${data.total_devices}`;
        document.getElementById('statApps').innerText = data.app_count;
        const tabBadge = document.getElementById('tabAppCountBadge');
        if (tabBadge) tabBadge.innerText = data.app_count;

        const latencyBadge = document.getElementById('latencyBadge');
        if (latencyBadge) {
            latencyBadge.innerText = t('directConnect', { ms: latency });
        }

        const lockConsoleBtn = document.getElementById('lockConsoleBtn');
        const pinProtectionBadge = document.getElementById('pinProtectionBadge');
        if (data.pin_required) {
            if (lockConsoleBtn) lockConsoleBtn.classList.remove('hidden');
            if (pinProtectionBadge) {
                pinProtectionBadge.innerHTML = '<i data-lucide="lock" class="w-4 h-4 text-emerald-500"></i><span class="text-emerald-600 dark:text-emerald-400">' + t('pinProtected') + '</span>';
            }
        } else {
            if (lockConsoleBtn) lockConsoleBtn.classList.add('hidden');
            if (pinProtectionBadge) {
                pinProtectionBadge.innerHTML = '<i data-lucide="unlock" class="w-4 h-4 text-slate-400"></i><span>' + t('pinUnprotected') + '</span>';
            }
        }

        const badge = document.getElementById('kernelStatusBadge');
        if (data.kernel_dpi_ready) {
            badge.className = 'hidden md:flex items-center space-x-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-emerald-50 text-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-800';
            badge.innerHTML = '<span class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span><span>' + t('dpiRunning') + '</span>';
        } else {
            badge.className = 'hidden md:flex items-center space-x-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-amber-50 text-amber-700 dark:bg-amber-950/50 dark:text-amber-300 border border-amber-200 dark:border-amber-800';
            badge.innerHTML = '<span class="w-2 h-2 rounded-full bg-amber-500"></span><span>' + t('dpiFallback') + '</span>';
        }
    } catch (e) {
        console.error('Fetch status failed:', e);
        const latencyBadge = document.getElementById('latencyBadge');
        if (latencyBadge) {
            latencyBadge.innerText = t('offline');
            latencyBadge.className = 'text-[10px] px-1.5 py-0.5 rounded-md font-mono bg-rose-50 text-rose-600 dark:bg-rose-950/60 dark:text-rose-400';
        }
    }
}

async function fetchCategories() {
    try {
        const res = await authFetch('/api/apps');
        appState.categories = await res.json() || [];
    } catch (e) {
        console.error('Fetch apps failed:', e);
    }
}

async function fetchMembers() {
    try {
        const res = await authFetch('/api/members');
        appState.members = await res.json() || [];
        renderMembers();
    } catch (e) {
        console.error('Fetch members failed:', e);
    }
}

async function fetchDevices() {
    try {
        const res = await authFetch('/api/devices');
        appState.devices = await res.json() || [];
        renderDevices();
    } catch (e) {
        console.error('Fetch devices failed:', e);
    }
}

async function fetchSettings() {
    try {
        const res = await authFetch('/api/settings');
        appState.settings = await res.json();
        document.getElementById('settingGlobalEnable').checked = appState.settings.enabled;
        document.getElementById('settingSafeSearch').checked = appState.settings.enforce_safe_search;
        document.getElementById('settingBlockDoH').checked = appState.settings.block_doh_dot;
        document.getElementById('settingIsolateNew').checked = appState.settings.isolate_new_devices;
        document.getElementById('settingPinCode').value = appState.settings.pin_code || '';
        document.getElementById('settingCloudSyncEnable').checked = appState.settings.cloud_sync_enabled || false;
        document.getElementById('settingCloudWorkerURL').value = appState.settings.cloud_worker_url || '';
        document.getElementById('settingCloudSecret').value = appState.settings.cloud_device_secret || '';
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
        const quota = m.quota_minutes || 0;
        const used = m.used_minutes || 0;
        const percent = quota > 0 ? Math.min(100, Math.round((used / quota) * 100)) : 0;
        const isQuotaExceeded = quota > 0 && used >= quota;
        const isBonus = m.bonus_until && new Date(m.bonus_until) > new Date();

        let cardBorderClass = 'border-slate-200/80 dark:border-slate-700/80';
        if (m.is_locked) {
            cardBorderClass = 'border-locked';
        } else if (isBonus) {
            cardBorderClass = 'border-bonus';
        }

        let statusBadge = `<span class="px-2 py-0.5 rounded-full text-xs font-semibold bg-emerald-100 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400">${t('normalOnline')}</span>`;
        if (m.is_locked) {
            statusBadge = `<span class="px-2 py-0.5 rounded-full text-xs font-semibold bg-red-100 text-red-700 dark:bg-red-950/60 dark:text-red-400 flex items-center space-x-1"><i data-lucide="lock" class="w-3 h-3"></i><span>${t('locked')}</span></span>`;
        } else if (isBonus) {
            statusBadge = `<span class="px-2 py-0.5 rounded-full text-xs font-semibold bg-amber-100 text-amber-700 dark:bg-amber-950/60 dark:text-amber-400 flex items-center space-x-1"><i data-lucide="zap" class="w-3 h-3"></i><span>${t('bonusActive')}</span></span>`;
        } else if (isQuotaExceeded) {
            statusBadge = `<span class="px-2 py-0.5 rounded-full text-xs font-semibold bg-orange-100 text-orange-700 dark:bg-orange-950/60 dark:text-orange-400">${t('quotaExceeded')}</span>`;
        }

        const avatarMap = { boy: '👦', girl: '👧', student: '🧑‍🎓', child: '👶' };
        const avatarEmoji = avatarMap[m.avatar] || '👦';

        return `
            <div class="bg-white dark:bg-slate-800 rounded-2xl p-5 border ${cardBorderClass} shadow-sm flex flex-col justify-between space-y-4 transition-all duration-200">
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
                            <p class="text-xs text-slate-400 mt-0.5">${m.device_macs ? m.device_macs.length : 0} devices · ${m.blocked_app_ids ? m.blocked_app_ids.length : 0} apps blocked</p>
                        </div>
                    </div>
                    <button onclick="editMember('${m.id}')" class="p-2 rounded-xl text-slate-400 hover:text-emerald-600 hover:bg-emerald-50 dark:hover:bg-slate-700 transition" title="${t('editRules')}">
                        <i data-lucide="sliders" class="w-4 h-4"></i>
                    </button>
                </div>

                <!-- 时间段管控摘要 -->
                <div class="text-xs bg-slate-50 dark:bg-slate-900/50 p-2.5 rounded-xl border border-slate-100 dark:border-slate-700/60 flex items-center space-x-1.5">
                    <i data-lucide="clock" class="w-3.5 h-3.5 text-slate-400 flex-shrink-0"></i>
                    <div class="truncate text-[11px]">${(() => {
                        if (m.schedule && m.schedule.enabled && m.schedule.time_ranges && m.schedule.time_ranges.length > 0) {
                            const isBlock = (m.schedule.action === 'block' || !m.schedule.action);
                            const actionText = isBlock ? `🚫 ${t('blockSchedulePrefix')}` : `✅ ${t('allowSchedulePrefix')}`;
                            const actionColor = isBlock ? 'text-amber-600 dark:text-amber-400' : 'text-emerald-600 dark:text-emerald-400';
                            const rangesText = m.schedule.time_ranges.map(tr => `${tr.start_time}~${tr.end_time}`).join(', ');
                            let daysText = t('dayAll');
                            if (m.schedule.days && m.schedule.days.length > 0 && m.schedule.days.length < 7) {
                                const sortedDays = m.schedule.days.slice().sort((a, b) => a - b);
                                const dayMap = {
                                    0: t('wDaySun'), 1: t('wDayMon'), 2: t('wDayTue'),
                                    3: t('wDayWed'), 4: t('wDayThu'), 5: t('wDayFri'), 6: t('wDaySat')
                                };
                                if (JSON.stringify(sortedDays) === JSON.stringify([1, 2, 3, 4, 5])) {
                                    daysText = t('dayWorkday');
                                } else if (JSON.stringify(sortedDays) === JSON.stringify([0, 6])) {
                                    daysText = t('dayWeekend');
                                } else {
                                    daysText = sortedDays.map(d => dayMap[d]).join('');
                                }
                            }
                            return `<span class="${actionColor} font-semibold">${actionText}: ${rangesText} (${daysText})</span>`;
                        }
                        return `<span class="text-slate-400 font-normal">${t('scheduleSummaryAllDay')}</span>`;
                    })()}</div>
                </div>

                <div class="space-y-1.5 bg-slate-50 dark:bg-slate-900/40 p-3 rounded-xl border border-slate-100 dark:border-slate-700/50">
                    <div class="flex justify-between text-xs font-medium text-slate-500 dark:text-slate-400">
                        <span>${t('todayUsed')}</span>
                        <span><b>${used}</b> / ${quota > 0 ? quota + ' min' : t('unlimited')}</span>
                    </div>
                    <div class="w-full bg-slate-200 dark:bg-slate-700 h-2 rounded-full overflow-hidden">
                        <div class="h-full rounded-full transition-width ${percent > 90 ? 'bg-red-500' : percent > 70 ? 'bg-amber-500' : 'bg-emerald-600'}" style="width: ${percent}%"></div>
                    </div>
                </div>

                <div class="grid grid-cols-2 gap-2 pt-1">
                    ${m.is_locked ? `
                        <button onclick="unlockMember('${m.id}')" class="flex items-center justify-center space-x-1.5 py-2.5 rounded-xl bg-emerald-500 hover:bg-emerald-600 active:scale-95 text-white text-xs font-semibold shadow-sm transition">
                            <i data-lucide="unlock" class="w-3.5 h-3.5"></i>
                            <span>${t('btnUnlock')}</span>
                        </button>
                    ` : `
                        <button onclick="lockMember('${m.id}')" class="flex items-center justify-center space-x-1.5 py-2.5 rounded-xl bg-rose-500 hover:bg-rose-600 active:scale-95 text-white text-xs font-semibold shadow-sm transition">
                            <i data-lucide="lock" class="w-3.5 h-3.5"></i>
                            <span>${t('btnLock')}</span>
                        </button>
                    `}
                    <button onclick="openBonusModal('${m.id}', '${m.name}')" class="flex items-center justify-center space-x-1.5 py-2.5 rounded-xl bg-amber-500 hover:bg-amber-600 active:scale-95 text-white text-xs font-semibold shadow-sm transition">
                        <i data-lucide="plus-circle" class="w-3.5 h-3.5"></i>
                        <span>${t('btnBonus')}</span>
                    </button>
                </div>
            </div>
        `;
    }).join('');
}

// 渲染局域网设备表格
function renderDevices(filterKeyword = '') {
    const tbody = document.getElementById('devicesTableBody');
    if (!appState.devices || appState.devices.length === 0) {
        tbody.innerHTML = '<tr><td colspan="8" class="px-4 py-8 text-center text-slate-400">' + t('noDevices') + '</td></tr>';
        return;
    }

    const filtered = appState.devices.filter(d => {
        if (!filterKeyword) return true;
        const kw = filterKeyword.toLowerCase();
        return (d.hostname && d.hostname.toLowerCase().includes(kw)) ||
               (d.ip && d.ip.includes(kw)) ||
               (d.mac && d.mac.toLowerCase().includes(kw)) ||
               (d.vendor && d.vendor.toLowerCase().includes(kw));
    });

    if (filtered.length === 0) {
        tbody.innerHTML = '<tr><td colspan="8" class="px-4 py-8 text-center text-slate-400">' + t('noMatchingDevices') + '</td></tr>';
        return;
    }

    tbody.innerHTML = filtered.map(d => {
        let memberName = `<span class="text-slate-400 text-xs">${t('unassigned')}</span>`;
        const member = appState.members.find(m => m.id === d.member_id || (m.device_macs && m.device_macs.includes(d.mac)));
        if (member) {
            memberName = `<span class="px-2 py-0.5 rounded-md bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-300 font-medium text-xs">${member.name}</span>`;
        }

        const speedText = d.rx_rate >= 1024 * 1024 
            ? (d.rx_rate / (1024 * 1024)).toFixed(1) + ' MB/s' 
            : (d.rx_rate / 1024).toFixed(0) + ' KB/s';

        const statusBadge = d.is_locked
            ? `<span class="px-2 py-0.5 rounded-full text-xs font-semibold bg-red-100 text-red-700 dark:bg-red-950/60 dark:text-red-400 inline-flex items-center space-x-1"><i data-lucide="lock" class="w-3 h-3"></i><span>${t('locked')}</span></span>`
            : `<span class="px-2 py-0.5 rounded-full text-xs font-semibold bg-emerald-100 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400 inline-flex items-center space-x-1"><span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span><span>${t('normalOnline')}</span></span>`;

        return `
            <tr class="hover:bg-slate-50/50 dark:hover:bg-slate-750/50 transition">
                <td class="px-4 py-3 font-semibold text-slate-800 dark:text-slate-100 flex items-center space-x-2">
                    <span class="w-2 h-2 rounded-full ${d.online ? 'bg-emerald-500 animate-pulse' : 'bg-slate-300 dark:bg-slate-600'}"></span>
                    <span>${d.hostname || 'Unknown'}</span>
                </td>
                <td class="px-4 py-3 font-mono text-xs">${d.ip || '-'}</td>
                <td class="px-4 py-3 font-mono text-xs text-slate-500 dark:text-slate-400">${d.mac}</td>
                <td class="px-4 py-3 text-xs">${d.vendor || 'Generic'}</td>
                <td class="px-4 py-3 text-xs font-mono text-emerald-600 dark:text-emerald-400 font-semibold">${d.online ? speedText : '-'}</td>
                <td class="px-4 py-3">${memberName}</td>
                <td class="px-4 py-3 text-center">${statusBadge}</td>
                <td class="px-4 py-3 text-right">
                    <div class="flex items-center justify-end space-x-2">
                        <button onclick="openAssignModal('${d.mac}', '${(d.hostname || d.ip || d.mac).replace(/'/g, "\\'")}', '${member ? member.id : ''}')" class="text-emerald-600 hover:text-emerald-700 text-xs font-semibold hover:underline">
                            ${t('btnAssign')}
                        </button>
                        <span class="text-slate-300 dark:text-slate-600">|</span>
                        ${d.is_locked ? `
                            <button onclick="toggleDeviceLock('${d.mac}', true)" class="text-emerald-600 hover:text-emerald-700 text-xs font-semibold hover:underline flex items-center space-x-1">
                                <i data-lucide="unlock" class="w-3 h-3"></i>
                                <span>${t('btnUnlock')}</span>
                            </button>
                        ` : `
                            <button onclick="toggleDeviceLock('${d.mac}', false)" class="text-rose-500 hover:text-rose-600 text-xs font-semibold hover:underline flex items-center space-x-1">
                                <i data-lucide="lock" class="w-3 h-3"></i>
                                <span>${t('btnLock')}</span>
                            </button>
                        `}
                    </div>
                </td>
            </tr>
        `;
    }).join('');
}

function filterDevicesTable() {
    const input = document.getElementById('deviceSearchInput');
    renderDevices(input ? input.value.trim() : '');
}

// 渲染特征库管理 (DPI 应用管理 Tab)
function renderAppManagement(keyword = '') {
    const container = document.getElementById('appsManagementContainer');
    if (!container) return;

    if (!appState.categories || appState.categories.length === 0) {
        container.innerHTML = '<div class="text-center py-12 text-slate-400">' + t('loadingDpi') + '</div>';
        return;
    }

    const kw = keyword.toLowerCase();

    container.innerHTML = appState.categories.map(cat => {
        const catTitle = tDpiCategory(cat.class_zh, cat.class_name);
        const filteredApps = cat.apps.filter(app => {
            const localizedAppName = tDpiApp(app.name, app.id).toLowerCase();
            return !kw || app.name.toLowerCase().includes(kw) || localizedAppName.includes(kw);
        });
        if (kw && filteredApps.length === 0) return '';

        const appsHTML = filteredApps.map(app => {
            const appDisplayName = tDpiApp(app.name, app.id);
            return `
                <div class="flex items-center justify-between p-2.5 rounded-xl bg-slate-50 dark:bg-slate-900 border border-slate-200/60 dark:border-slate-700/60 hover:border-emerald-500 transition">
                    <div class="flex items-center space-x-2 truncate">
                        <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 flex-shrink-0"></span>
                        <span class="font-medium text-xs text-slate-800 dark:text-slate-100 truncate">${appDisplayName}</span>
                        <span class="text-[10px] text-slate-400 font-mono flex-shrink-0">#${app.id}</span>
                    </div>
                    <button onclick="deleteApp(${app.id})" class="text-slate-400 hover:text-rose-500 transition ml-1" title="Delete">
                        <i data-lucide="trash" class="w-3.5 h-3.5"></i>
                    </button>
                </div>
            `;
        }).join('');

        return `
            <div class="bg-white dark:bg-slate-800 rounded-2xl p-5 border border-slate-200 dark:border-slate-700 shadow-sm space-y-3">
                <div class="flex items-center justify-between border-b border-slate-100 dark:border-slate-700/60 pb-3">
                    <div class="flex items-center space-x-2.5">
                        <div class="w-7 h-7 rounded-lg bg-emerald-100 dark:bg-emerald-950/60 text-emerald-600 flex items-center justify-center">
                            <i data-lucide="${cat.icon || 'grid'}" class="w-4 h-4"></i>
                        </div>
                        <h4 class="font-bold text-sm text-slate-800 dark:text-white">${catTitle}</h4>
                        <span class="text-xs text-slate-400 font-mono">(${filteredApps.length})</span>
                    </div>
                    <div class="flex items-center space-x-2">
                        <button onclick="openAppModal(${cat.class_id})" class="text-xs text-emerald-600 hover:text-emerald-700 font-semibold flex items-center space-x-1">
                            <i data-lucide="plus" class="w-3.5 h-3.5"></i>
                            <span>${t('btnAddApp')}</span>
                        </button>
                    </div>
                </div>
                <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-2">
                    ${appsHTML}
                </div>
            </div>
        `;
    }).join('');

    lucide.createIcons();
}

function filterAppsGrid() {
    const input = document.getElementById('appFilterInput');
    renderAppManagement(input ? input.value.trim() : '');
}

// 应用与分类创建 Modal
function openAppModal(classId = null) {
    const modal = document.getElementById('appModal');
    const select = document.getElementById('formAppCategory');
    select.innerHTML = appState.categories.map(c => `
        <option value="${c.class_id}" ${classId === c.class_id ? 'selected' : ''}>${tDpiCategory(c.class_zh, c.class_name)}</option>
    `).join('');

    document.getElementById('formAppId').value = '';
    document.getElementById('formAppName').value = '';
    document.getElementById('formAppDescription').value = '';
    modal.classList.remove('hidden');
    lucide.createIcons();
}

function closeAppModal() {
    document.getElementById('appModal').classList.add('hidden');
}

async function saveAppForm() {
    const name = document.getElementById('formAppName').value.trim();
    const classId = parseInt(document.getElementById('formAppCategory').value);
    if (!name) {
        showToast(t('toastInputAppName'), 'warning');
        return;
    }

    const payload = {
        name: name,
        class_id: classId
    };

    try {
        const res = await authFetch('/api/apps', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        if (res.ok) {
            closeAppModal();
            showToast(t('toastAppAdded'), 'success');
            await fetchCategories();
            renderAppManagement();
        } else {
            showToast(t('toastFailed'), 'error');
        }
    } catch (e) {
        showToast(t('toastNetworkError'), 'error');
    }
}

async function deleteApp(appId) {
    if (!confirm(t('confirmDeleteApp'))) return;
    try {
        const res = await authFetch(`/api/apps/${appId}`, { method: 'DELETE' });
        if (res.ok) {
            showToast(t('toastAppDeleted'), 'info');
            await fetchCategories();
            renderAppManagement();
        } else {
            showToast(t('toastFailed'), 'error');
        }
    } catch (e) {
        showToast(t('toastNetworkError'), 'error');
    }
}

function openCategoryModal() {
    document.getElementById('formCategoryName').value = '';
    document.getElementById('categoryModal').classList.remove('hidden');
    lucide.createIcons();
}

function closeCategoryModal() {
    document.getElementById('categoryModal').classList.add('hidden');
}

async function saveCategoryForm() {
    const name = document.getElementById('formCategoryName').value.trim();
    const icon = document.getElementById('formCategoryIcon').value;
    if (!name) {
        showToast(t('toastInputCategoryName'), 'warning');
        return;
    }

    const payload = {
        name: name,
        icon: icon
    };

    try {
        const res = await authFetch('/api/categories', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        if (res.ok) {
            closeCategoryModal();
            showToast(t('toastCategoryCreated'), 'success');
            await fetchCategories();
            renderAppManagement();
        } else {
            showToast(t('toastFailed'), 'error');
        }
    } catch (e) {
        showToast(t('toastNetworkError'), 'error');
    }
}

// 加时弹窗控制
function openBonusModal(id, name) {
    document.getElementById('bonusMemberId').value = id;
    document.getElementById('bonusMemberName').innerText = name;
    document.getElementById('bonusModal').classList.remove('hidden');
    lucide.createIcons();
}

function closeBonusModal() {
    document.getElementById('bonusModal').classList.add('hidden');
}

async function applyBonusTime(minutes) {
    const id = document.getElementById('bonusMemberId').value;
    if (!id) return;
    closeBonusModal();

    try {
        const res = await authFetch(`/api/members/${id}/bonus?minutes=${minutes}`, { method: 'POST' });
        if (res.ok) {
            showToast(t('toastBonusGranted', { min: minutes }), 'success');
            await fetchMembers();
            lucide.createIcons();
        } else {
            showToast(t('toastFailed'), 'error');
        }
    } catch (e) {
        showToast(t('toastNetworkError'), 'error');
    }
}

// 星期选择快捷操作
function selectScheduleDays(preset) {
    const checkboxes = document.querySelectorAll('input[name="scheduleDay"]');
    checkboxes.forEach(cb => {
        const val = parseInt(cb.value);
        if (preset === 'all') {
            cb.checked = true;
        } else if (preset === 'workday') {
            cb.checked = val >= 1 && val <= 5;
        } else if (preset === 'weekend') {
            cb.checked = val === 6 || val === 0;
        }
    });
}

// 快捷预设（夜间防沉迷 / 上学日管控）
function applyPresetSchedule(preset) {
    const container = document.getElementById('modalTimeRangeList');
    if (!container) return;
    container.innerHTML = '';

    if (preset === 'night') {
        const blockRadio = document.querySelector('input[name="scheduleAction"][value="block"]');
        if (blockRadio) blockRadio.checked = true;
        selectScheduleDays('all');
        addTimeRangeRow('21:30', '07:00');
    } else if (preset === 'school') {
        const blockRadio = document.querySelector('input[name="scheduleAction"][value="block"]');
        if (blockRadio) blockRadio.checked = true;
        selectScheduleDays('workday');
        addTimeRangeRow('08:00', '11:30');
        addTimeRangeRow('14:00', '17:30');
        addTimeRangeRow('21:30', '07:00');
    }
}

// 动态添加时间段行
function addTimeRangeRow(startTime = '21:30', endTime = '07:00') {
    const container = document.getElementById('modalTimeRangeList');
    if (!container) return;

    const currentCount = container.querySelectorAll('.time-range-row').length;
    const row = document.createElement('div');
    row.className = 'time-range-row flex items-center space-x-2 bg-white dark:bg-slate-800 p-2 rounded-xl border border-slate-200 dark:border-slate-700';
    row.innerHTML = `
        <div class="flex items-center space-x-1.5 flex-1">
            <span class="text-xs font-mono font-bold text-slate-400 w-5 text-center slot-index">${currentCount + 1}</span>
            <input type="time" class="time-range-start px-2 py-1.5 rounded-lg bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 text-xs font-mono text-slate-800 dark:text-slate-100" value="${startTime}">
            <span class="text-slate-400 text-xs">${t('timeRangeTo')}</span>
            <input type="time" class="time-range-end px-2 py-1.5 rounded-lg bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 text-xs font-mono text-slate-800 dark:text-slate-100" value="${endTime}">
        </div>
        <button type="button" onclick="removeTimeRangeRow(this)" class="p-1.5 text-slate-400 hover:text-rose-500 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700 transition" title="Delete">
            <i data-lucide="trash-2" class="w-3.5 h-3.5"></i>
        </button>
    `;
    container.appendChild(row);
    lucide.createIcons();
}

function removeTimeRangeRow(btn) {
    const row = btn.closest('.time-range-row');
    if (row) {
        row.remove();
        // 重新编号
        document.querySelectorAll('#modalTimeRangeList .time-range-row').forEach((r, idx) => {
            const numEl = r.querySelector('.slot-index');
            if (numEl) numEl.innerText = idx + 1;
        });
    }
}

// 成员编辑与创建 Modal 逻辑
function openMemberModal(member = null) {
    const modal = document.getElementById('memberModal');
    const title = document.getElementById('modalTitle');
    const btnDel = document.getElementById('btnDeleteMember');
    const timeRangeContainer = document.getElementById('modalTimeRangeList');
    if (timeRangeContainer) timeRangeContainer.innerHTML = '';

    if (member) {
        title.innerText = t('editMemberTitle', { name: member.name });
        document.getElementById('formMemberId').value = member.id;
        document.getElementById('formMemberName').value = member.name;
        document.getElementById('formMemberAvatar').value = member.avatar || 'boy';
        document.getElementById('formQuotaMinutes').value = member.quota_minutes || '';

        // 回显时间表
        const schedule = member.schedule || { enabled: true, days: [0, 1, 2, 3, 4, 5, 6], time_ranges: [], action: 'block' };
        document.getElementById('formScheduleEnable').checked = schedule.enabled !== false;

        // 回显动作模式
        const action = schedule.action || 'block';
        const actionRadio = document.querySelector(`input[name="scheduleAction"][value="${action}"]`);
        if (actionRadio) actionRadio.checked = true;

        // 回显星期
        const daysSet = new Set(schedule.days || [0, 1, 2, 3, 4, 5, 6]);
        document.querySelectorAll('input[name="scheduleDay"]').forEach(cb => {
            cb.checked = daysSet.has(parseInt(cb.value));
        });

        // 回显多个时间段
        if (schedule.time_ranges && schedule.time_ranges.length > 0) {
            schedule.time_ranges.forEach(tr => {
                addTimeRangeRow(tr.start_time || '21:30', tr.end_time || '07:00');
            });
        } else {
            addTimeRangeRow('21:30', '07:00');
        }

        btnDel.classList.remove('hidden');
    } else {
        title.innerText = t('addMemberTitle');
        document.getElementById('formMemberId').value = '';
        document.getElementById('formMemberName').value = '';
        document.getElementById('formMemberAvatar').value = 'boy';
        document.getElementById('formQuotaMinutes').value = '120';
        document.getElementById('formScheduleEnable').checked = true;

        const defaultActionRadio = document.querySelector('input[name="scheduleAction"][value="block"]');
        if (defaultActionRadio) defaultActionRadio.checked = true;

        selectScheduleDays('all');
        addTimeRangeRow('21:30', '07:00');

        btnDel.classList.add('hidden');
    }

    toggleScheduleForm();
    renderModalDevices(member ? member.device_macs : []);
    renderModalAppCategories(member ? member.blocked_app_ids : []);

    modal.classList.remove('hidden');
    lucide.createIcons();
}

function toggleScheduleForm() {
    const isEnabled = document.getElementById('formScheduleEnable')?.checked;
    const content = document.getElementById('scheduleDetailsBlock');
    if (!content) return;
    if (isEnabled) {
        content.classList.remove('opacity-40', 'pointer-events-none');
    } else {
        content.classList.add('opacity-40', 'pointer-events-none');
    }
}

function closeMemberModal() {
    document.getElementById('memberModal').classList.add('hidden');
}

function renderModalDevices(selectedMACs = []) {
    const container = document.getElementById('modalDeviceList');
    if (!appState.devices || appState.devices.length === 0) {
        container.innerHTML = '<div class="text-xs text-slate-400 p-2">' + t('noDevicesDetected') + '</div>';
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
        container.innerHTML = '<div class="text-xs text-slate-400 p-2">' + t('loadingDpi') + '</div>';
        return;
    }

    container.innerHTML = appState.categories.map(cat => {
        const catTitle = tDpiCategory(cat.class_zh, cat.class_name);
        const appsHTML = cat.apps.map(app => {
            const isChecked = selectedSet.has(app.id);
            const appDisplayName = tDpiApp(app.name, app.id);
            return `
                <label class="inline-flex items-center space-x-1.5 bg-white dark:bg-slate-800 px-2.5 py-1.5 rounded-lg border border-slate-200 dark:border-slate-700 cursor-pointer text-xs">
                    <input type="checkbox" name="modalApp" value="${app.id}" ${isChecked ? 'checked' : ''} onchange="updateSelectedCount()" class="w-3.5 h-3.5 text-emerald-600 rounded">
                    <span>${appDisplayName}</span>
                </label>
            `;
        }).join('');

        return `
            <div class="border border-slate-200 dark:border-slate-700 rounded-xl p-2.5 bg-slate-50/50 dark:bg-slate-900/30">
                <div class="flex items-center justify-between mb-2">
                    <span class="font-bold text-xs text-slate-700 dark:text-slate-200 flex items-center space-x-1">
                        <span>${catTitle}</span>
                    </span>
                    <button type="button" onclick="toggleSelectAllCategory(${cat.class_id})" class="text-[11px] text-emerald-600 hover:underline">${t('btnToggleSelect')}</button>
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

// 设备一键断网 / 恢复上网
async function toggleDeviceLock(mac, currentlyLocked) {
    const action = currentlyLocked ? 'unlock' : 'lock';
    try {
        const res = await authFetch(`/api/devices/${mac}/${action}`, { method: 'POST' });
        if (res.ok) {
            showToast(currentlyLocked ? t('toastUnlocked') : t('toastLocked'), 'info');
            await Promise.all([fetchStatus(), fetchMembers(), fetchDevices()]);
            lucide.createIcons();
        } else {
            showToast(t('toastFailed'), 'error');
        }
    } catch (e) {
        showToast(t('toastNetworkError'), 'error');
    }
}

// 设备分配成员 Modal
function openAssignModal(mac, deviceName, currentMemberId) {
    document.getElementById('assignDeviceMAC').value = mac;
    document.getElementById('assignModalSubtitle').innerText = `${t('colDeviceName')}: ${deviceName} (${mac})`;

    const container = document.getElementById('assignMemberList');
    const avatarMap = { boy: '👦', girl: '👧', student: '🧑‍🎓', child: '👶' };

    let html = `
        <label class="flex items-center justify-between p-3 rounded-2xl border ${!currentMemberId ? 'border-emerald-500 bg-emerald-50/50 dark:bg-emerald-950/30' : 'border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800'} cursor-pointer hover:border-emerald-500 transition">
            <div class="flex items-center space-x-3">
                <input type="radio" name="assignMemberRadio" value="" ${!currentMemberId ? 'checked' : ''} class="w-4 h-4 text-emerald-600">
                <div class="w-8 h-8 rounded-xl bg-slate-100 dark:bg-slate-700 flex items-center justify-center text-slate-500 text-sm">
                    🚫
                </div>
                <div>
                    <div class="font-bold text-xs text-slate-800 dark:text-slate-100">${t('unassigned')}</div>
                    <div class="text-[11px] text-slate-400">${t('unbindDeviceDesc')}</div>
                </div>
            </div>
        </label>
    `;

    if (appState.members && appState.members.length > 0) {
        html += appState.members.map(m => {
            const isSelected = m.id === currentMemberId;
            const emoji = avatarMap[m.avatar] || '👦';
            const devCount = m.device_macs ? m.device_macs.length : 0;
            return `
                <label class="flex items-center justify-between p-3 rounded-2xl border ${isSelected ? 'border-emerald-500 bg-emerald-50/50 dark:bg-emerald-950/30' : 'border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800'} cursor-pointer hover:border-emerald-500 transition">
                    <div class="flex items-center space-x-3">
                        <input type="radio" name="assignMemberRadio" value="${m.id}" ${isSelected ? 'checked' : ''} class="w-4 h-4 text-emerald-600">
                        <div class="w-8 h-8 rounded-xl bg-emerald-50 dark:bg-emerald-950/60 flex items-center justify-center text-base">
                            ${emoji}
                        </div>
                        <div>
                            <div class="font-bold text-xs text-slate-800 dark:text-slate-100">${m.name}</div>
                            <div class="text-[11px] text-slate-400">${t('statDevices')}: ${devCount} · ${t('todayUsed')}: ${m.quota_minutes || 0} min</div>
                        </div>
                    </div>
                </label>
            `;
        }).join('');
    }

    container.innerHTML = html;
    document.getElementById('assignModal').classList.remove('hidden');
    lucide.createIcons();
}

function closeAssignModal() {
    document.getElementById('assignModal').classList.add('hidden');
}

function openNewMemberFromAssign() {
    const mac = document.getElementById('assignDeviceMAC').value;
    closeAssignModal();
    openMemberModal();
    if (mac) {
        const cb = document.querySelector(`input[name="modalDevice"][value="${mac}"]`);
        if (cb) cb.checked = true;
    }
}

async function submitDeviceAssign() {
    const mac = document.getElementById('assignDeviceMAC').value;
    if (!mac) return;

    const selectedRadio = document.querySelector('input[name="assignMemberRadio"]:checked');
    const memberId = selectedRadio ? selectedRadio.value : '';

    try {
        const res = await authFetch(`/api/devices/${mac}/assign`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ member_id: memberId })
        });
        if (res.ok) {
            closeAssignModal();
            showToast(t('toastDeviceAssigned'), 'success');
            await Promise.all([fetchMembers(), fetchDevices()]);
            lucide.createIcons();
        } else {
            showToast(t('toastFailed'), 'error');
        }
    } catch (e) {
        showToast(t('toastNetworkError'), 'error');
    }
}

function quickAssignDevice(mac) {
    openAssignModal(mac, mac, '');
}

function editMember(id) {
    const member = appState.members.find(m => m.id === id);
    if (member) {
        openMemberModal(member);
    }
}

async function saveMemberForm() {
    const id = document.getElementById('formMemberId').value || 'm_' + Date.now();
    const name = document.getElementById('formMemberName').value.trim();
    if (!name) {
        showToast(t('toastInputMemberName'), 'warning');
        return;
    }

    const avatar = document.getElementById('formMemberAvatar').value;
    const quota = parseInt(document.getElementById('formQuotaMinutes').value) || 0;
    
    const scheduleEnabled = document.getElementById('formScheduleEnable').checked;
    const scheduleAction = document.querySelector('input[name="scheduleAction"]:checked')?.value || 'block';
    const scheduleDays = Array.from(document.querySelectorAll('input[name="scheduleDay"]:checked')).map(cb => parseInt(cb.value));

    const timeRanges = [];
    document.querySelectorAll('.time-range-row').forEach(row => {
        const start = row.querySelector('.time-range-start')?.value || '';
        const end = row.querySelector('.time-range-end')?.value || '';
        if (start && end) {
            timeRanges.push({ start_time: start, end_time: end });
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
            days: scheduleDays.length > 0 ? scheduleDays : [0, 1, 2, 3, 4, 5, 6],
            time_ranges: timeRanges,
            action: scheduleAction
        },
        blocked_app_ids: blockedAppIDs,
        safe_search: true,
        block_adult: true
    };

    try {
        const res = await authFetch('/api/members', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        if (res.ok) {
            closeMemberModal();
            showToast(t('toastRuleSaved'), 'success');
            await fetchMembers();
            lucide.createIcons();
        } else {
            showToast(t('toastFailed'), 'error');
        }
    } catch (e) {
        showToast(t('toastNetworkError'), 'error');
    }
}

async function deleteCurrentMember() {
    const id = document.getElementById('formMemberId').value;
    if (!id || !confirm(t('confirmDeleteMember'))) return;

    try {
        await authFetch(`/api/members/${id}`, { method: 'DELETE' });
        closeMemberModal();
        showToast(t('toastMemberDeleted'), 'info');
        await fetchMembers();
        lucide.createIcons();
    } catch (e) {
        showToast(t('toastFailed'), 'error');
    }
}

async function lockMember(id) {
    try {
        const res = await authFetch(`/api/members/${id}/lock`, { method: 'POST' });
        if (res.ok) {
            showToast(t('toastLocked'), 'warning');
            await fetchMembers();
            lucide.createIcons();
        }
    } catch (e) {
        showToast(t('toastFailed'), 'error');
    }
}

async function unlockMember(id) {
    try {
        const res = await authFetch(`/api/members/${id}/unlock`, { method: 'POST' });
        if (res.ok) {
            showToast(t('toastUnlocked'), 'success');
            await fetchMembers();
            lucide.createIcons();
        }
    } catch (e) {
        showToast(t('toastFailed'), 'error');
    }
}

function clearPinSetting() {
    document.getElementById('settingPinCode').value = '';
}

async function saveGlobalSettings() {
    const pinVal = document.getElementById('settingPinCode').value.trim();
    if (pinVal && (!/^\d{4}$/.test(pinVal))) {
        showToast(t('toastPinLengthError'), 'warning');
        return;
    }

    const payload = {
        enabled: document.getElementById('settingGlobalEnable').checked,
        pin_code: pinVal,
        cloud_sync_enabled: document.getElementById('settingCloudSyncEnable').checked,
        cloud_worker_url: document.getElementById('settingCloudWorkerURL').value.trim(),
        cloud_device_secret: document.getElementById('settingCloudSecret').value.trim(),
        enforce_safe_search: document.getElementById('settingSafeSearch').checked,
        block_doh_dot: document.getElementById('settingBlockDoH').checked,
        isolate_new_devices: document.getElementById('settingIsolateNew').checked,
    };

    try {
        const res = await authFetch('/api/settings', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        if (res.ok) {
            if (pinVal) {
                setStoredPin(pinVal);
            } else {
                setStoredPin('');
            }
            showToast(t('toastSettingsSaved'), 'success');
            await fetchStatus();
        }
    } catch (e) {
        showToast(t('toastFailed'), 'error');
    }
}
