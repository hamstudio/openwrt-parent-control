'use strict';
'require view';
'require dom';
'require poll';
'require rpc';
'require fs';
'require ui';

var callServiceList = rpc.declare({
	object: 'service',
	method: 'list',
	params: [ 'name' ],
	expect: { 'parentcontrol': {} }
});

return view.extend({
	load: function() {
		return Promise.all([
			callServiceList('parentcontrol').catch(function() { return {}; }),
			fs.read_direct('/etc/parentcontrol/config.json', 'json').catch(function() { return null; })
		]);
	},

	render: function(data) {
		var serviceInfo = data[0] || {};
		var configData = data[1] || {};
		var isRunning = false;

		if (serviceInfo && serviceInfo.instances) {
			for (var instance in serviceInfo.instances) {
				if (serviceInfo.instances[instance].running) {
					isRunning = true;
					break;
				}
			}
		}

		var host = window.location.hostname;
		var isHttps = (window.location.protocol === 'https:');
		var basePort = (configData && configData.settings && configData.settings.web_port) ? configData.settings.web_port : 8088;
		var port = isHttps ? (basePort + 1) : basePort;
		var dashboardUrl = (isHttps ? 'https://' : 'http://') + host + ':' + port;
		var httpUrl = 'http://' + host + ':' + basePort;

		var statusBadge = isRunning 
			? E('span', { 'class': 'badge', 'style': 'background:#10b981; color:#fff; padding:3px 8px; border-radius:4px; font-weight:bold;' }, _('运行中 (Running)'))
			: E('span', { 'class': 'badge', 'style': 'background:#ef4444; color:#fff; padding:3px 8px; border-radius:4px; font-weight:bold;' }, _('未运行 (Stopped)'));

		var viewRoot = E('div', { 'class': 'cbi-map' }, [
			E('h2', { 'style': 'margin-bottom:4px;' }, _('家长控制与应用安全管控系统 (ParentControl Guard)')),
			E('div', { 'class': 'cbi-map-descr', 'style': 'margin-bottom:16px;' }, _('基于 kmod-oaf 七层深度包检测 (DPI)、时间配额调度与 Cloudflare Worker 公网同步。')),

			// 状态与快捷操作卡片
			E('div', { 'class': 'cbi-section' }, [
				E('div', { 'class': 'cbi-section-node' }, [
					E('div', { 'class': 'cbi-value' }, [
						E('label', { 'class': 'cbi-value-title' }, _('服务运行状态')),
						E('div', { 'class': 'cbi-value-field', 'id': 'pc-status-field' }, [ statusBadge ])
					]),
					E('div', { 'class': 'cbi-value' }, [
						E('label', { 'class': 'cbi-value-title' }, _('控制台地址')),
						E('div', { 'class': 'cbi-value-field' }, [
							E('a', {
								'class': 'btn cbi-button cbi-button-apply',
								'href': dashboardUrl,
								'target': '_blank',
								'style': 'background:#059669; color:#fff; padding:6px 14px; border-radius:6px; text-decoration:none; font-weight:bold; display:inline-block; margin-right:8px;'
							}, [ _('↗ 新窗口全屏打开控制台 (%s)').format(isHttps ? 'HTTPS :8089' : 'HTTP :8088') ]),
							E('a', {
								'class': 'btn cbi-button cbi-button-neutral',
								'href': httpUrl,
								'target': '_blank',
								'style': 'padding:6px 12px; border-radius:6px; text-decoration:none; display:inline-block;'
							}, [ _('HTTP 直连 (:8088)') ]),
							E('div', { 'class': 'cbi-value-description', 'style': 'margin-top:6px;' }, _('已自动启用与系统一致的 SSL 证书加密。支持移动端与桌面端自适应、4位数密码访问控制。'))
						])
					])
				])
			]),

			// 内嵌控制台与 SSL 信任指引
			E('div', { 'class': 'cbi-section', 'style': 'margin-top:16px;' }, [
				E('div', { 'style': 'display:flex; justify-content:space-between; align-items:center; margin-bottom:8px;' }, [
					E('h3', { 'style': 'margin:0;' }, _('内嵌控制面板')),
					isHttps ? E('div', { 'class': 'text-xs', 'style': 'font-size:12px; color:#64748b;' }, [
						_('若内嵌显示证书错误，请先 '),
						E('a', {
							'href': dashboardUrl,
							'target': '_blank',
							'style': 'color:#059669; font-weight:bold; text-decoration:underline;'
						}, [ _('点击这里在新标签页信任证书') ]),
						_(' 或直接使用上方直达按钮')
					]) : E('span', {})
				]),
				E('div', {
					'style': 'width:100%; border:1px solid #cbd5e1; border-radius:8px; overflow:hidden; background:#fff; box-shadow:0 2px 4px rgba(0,0,0,0.05);'
				}, [
					E('iframe', {
						'src': dashboardUrl,
						'style': 'width:100%; height:820px; border:none; display:block;',
						'id': 'pc-dashboard-iframe'
					})
				])
			])
		]);

		return viewRoot;
	},

	handleSaveApply: null,
	handleSave: null,
	handleReset: null
});
