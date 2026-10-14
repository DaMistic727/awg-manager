import type { Subscription } from '$lib/types';

export function resolveSubscriptionMemberTag(
	subscription: Subscription,
	liveActiveMember?: string | null,
): string {
	if (liveActiveMember && subscription.memberTags.includes(liveActiveMember)) {
		return liveActiveMember;
	}
	if (subscription.activeMember && subscription.memberTags.includes(subscription.activeMember)) {
		return subscription.activeMember;
	}
	return subscription.memberTags[0] ?? '';
}

/**
 * subscriptionName — показываемое имя подписки: подпись, иначе URL, иначе
 * путь файла, иначе id. У файловой подписки url пуст — без path заголовок и
 * подтверждение удаления остались бы безымянными (F431).
 */
export function subscriptionName(s: Pick<Subscription, 'id' | 'label' | 'url' | 'path'>): string {
	return s.label || s.url || s.path || s.id;
}
