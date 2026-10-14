import { describe, it, expect } from 'vitest';
import { subscriptionName } from './subscriptionMember';

// F431: у файловой подписки url пуст — имя берётся из пути, а не пустеет.
describe('subscriptionName', () => {
	it('подпись, иначе URL, иначе путь, иначе id', () => {
		expect(subscriptionName({ id: 'a', label: 'L', url: 'u', path: 'p' })).toBe('L');
		expect(subscriptionName({ id: 'a', label: '', url: 'u', path: '' })).toBe('u');
		expect(subscriptionName({ id: 'a', label: '', url: '', path: '/opt/sub.txt' })).toBe('/opt/sub.txt');
		expect(subscriptionName({ id: 'a', label: '', url: '', path: '' })).toBe('a');
	});
});
