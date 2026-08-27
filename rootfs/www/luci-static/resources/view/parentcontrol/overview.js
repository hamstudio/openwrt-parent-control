'use strict';
'require view';
'require dom';
'require fs';
'require ui';

return view.extend({
	render: function() {
		var host = window.location.hostname;
		var dashboardUrl = 'http://' + host + ':8088';

		var body = E('div', { 'class': 'cbi-map' }, [
			E('h2', {}, _('家长控制与应用安全管控系统')),
			E('div', { 'class': 'cbi-map-descr' }, _('基于 kmod-oaf 深度包检测 (DPI) 与细粒度时间配额引擎。')),
			
			E('div', { 'class': 'cbi-section' }, [
				E('div', { 'class': 'cbi-section-node' }, [
					E('div', { 'class': 'cbi-value' }, [
						E('label', { 'class': 'cbi-value-title' }, _('独立控制台')),
						E('div', { 'class': 'cbi-value-field' }, [
							E('a', {
								'class': 'btn cbi-button cbi-button-apply',
								'href': dashboardUrl,
								'target': '_blank',
								'style': 'background-color: #059669; color: white; padding: 8px 16px; border-radius: 8px; text-decoration: none; display: inline-block; font-weight: bold;'
							}, [
								_('🌿 打开绿色健康守护控制台 (端口: 8088)')
							]),
							E('div', { 'class': 'cbi-value-description' }, _('点击可在新标签页中打开适配手机与电脑端的家长控制中心，支持一键断网、时间配额滑块及数百款热门 App 精准封禁。'))
						])
					]),
					E('div', { 'class': 'cbi-value' }, [
						E('label', { 'class': 'cbi-value-title' }, _('服务状态')),
						E('div', { 'class': 'cbi-value-field' }, [
							E('span', { 'class': 'badge', 'id': 'pc-status-badge', 'style': 'padding: 4px 8px; background: #10b981; color: white; border-radius: 4px;' }, _('运行中'))
						])
					])
				])
			])
		]);

		return body;
	},

	handleSaveApply: null,
	handleSave: null,
	handleReset: null
});
