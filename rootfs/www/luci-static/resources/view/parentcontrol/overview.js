'use strict';
'require view';
'require dom';
'require fs';
'require ui';

return view.extend({
	render: function() {
		var host = window.location.hostname;
		var dashboardUrl = window.location.protocol + '//' + host + ':8088';

		var body = E('div', { 'class': 'cbi-map' }, [
			E('div', { 'style': 'display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px;' }, [
				E('div', {}, [
					E('h2', { 'style': 'margin-bottom: 4px;' }, _('家长控制中心 (ParentControl Guard)')),
					E('div', { 'class': 'cbi-map-descr', 'style': 'margin-bottom: 0;' }, _('基于 kmod-oaf 深度包检测 (DPI) 与细粒度时间配额引擎'))
				]),
				E('div', {}, [
					E('a', {
						'class': 'btn cbi-button cbi-button-apply',
						'href': dashboardUrl,
						'target': '_blank',
						'style': 'background-color: #059669; color: white; padding: 6px 14px; border-radius: 8px; text-decoration: none; display: inline-flex; align-items: center; font-weight: bold; box-shadow: 0 2px 4px rgba(5, 150, 105, 0.2);'
					}, [
						_('↗ 新窗口全屏打开控制台')
					])
				])
			]),

			// 内嵌完整控制台视图 (无缝在 LuCI 内部操作)
			E('div', {
				'class': 'cbi-section',
				'style': 'border-radius: 12px; overflow: hidden; box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1); border: 1px solid #e2e8f0; margin-top: 10px; background: #fff;'
			}, [
				E('iframe', {
					'src': dashboardUrl,
					'style': 'width: 100%; height: 780px; border: none; display: block;',
					'id': 'parentcontrol-iframe'
				})
			])
		]);

		return body;
	},

	handleSaveApply: null,
	handleSave: null,
	handleReset: null
});
