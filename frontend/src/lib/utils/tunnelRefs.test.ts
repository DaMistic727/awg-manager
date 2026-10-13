import { describe, expect, it } from 'vitest';
import { describeRouterReference } from './tunnelRefs';

describe('describeRouterReference', () => {
	it('describes route.final', () => {
		expect(describeRouterReference('route.final')).toEqual({
			text: 'TProxy → Назначен маршрутом по умолчанию',
			known: true,
			plane: 'tproxy'
		});
	});

	it('describes a composite member', () => {
		expect(describeRouterReference('outbounds[0="vpn-default"].outbounds[3]')).toEqual({
			text: 'TProxy → Входит в группу маршрутов «vpn-default»',
			known: true,
			plane: 'tproxy'
		});
	});

	it('describes a composite default', () => {
		expect(describeRouterReference('outbounds[2="grp"].default')).toEqual({
			text: 'TProxy → Выбран по умолчанию в группе «grp»',
			known: true,
			plane: 'tproxy'
		});
	});

	it('describes a dns detour', () => {
		expect(describeRouterReference('dns.servers[1="my-dns"].detour')).toEqual({
			text: 'TProxy → Используется DNS-сервером «my-dns»',
			known: true,
			plane: 'tproxy'
		});
	});

	it('describes a rule_set download_detour', () => {
		expect(describeRouterReference('route.rule_set[0="geoip-ru"].download_detour')).toEqual({
			text: 'TProxy → Через него скачивается список «geoip-ru»',
			known: true,
			plane: 'tproxy'
		});
	});

	it('handles names with special characters', () => {
		expect(describeRouterReference('outbounds[0="vpn [eu] #1"].outbounds[0]')).toEqual({
			text: 'TProxy → Входит в группу маршрутов «vpn [eu] #1»',
			known: true,
			plane: 'tproxy'
		});
	});

	it('describes a fakeip rule with the 0-based number its table shows', () => {
		expect(describeRouterReference('[fakeip] route.rules[9]')).toEqual({
			text: 'FakeIP → Используется в правиле #9',
			known: true,
			plane: 'fakeip'
		});
	});

	it('describes a fakeip non-rule location', () => {
		expect(describeRouterReference('[fakeip] route.final')).toEqual({
			text: 'FakeIP → Назначен маршрутом по умолчанию',
			known: true,
			plane: 'fakeip'
		});
	});

	it('falls back to the raw path for unknown formats', () => {
		expect(describeRouterReference('something.unexpected[5]')).toEqual({
			text: 'something.unexpected[5]',
			known: false,
			plane: 'tproxy'
		});
	});
});
