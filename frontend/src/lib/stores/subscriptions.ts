import { api } from '$lib/api/client';
import { createPollingStore } from './polling';
import { registerStore } from './storeRegistry';
import type { Subscription, SubscriptionGroup } from '$lib/types';

// Подписки обновляет событие: бэкенд публикует singbox.subscriptions на
// каждую запись стора — REST, MCP и фоновое обновление (MC9). Таймер
// оставлен страховкой: invalidate во время идущего запроса теряется (F663),
// а обновление подписки пишет стор несколько раз подряд.
export const subscriptionsStore = createPollingStore<Subscription[]>(
	() => api.listSubscriptions(),
	{ staleTime: 5_000, pollInterval: 30_000 },
);
registerStore('singbox.subscriptions', subscriptionsStore);

// #572: сводные группы для резолва agg-тегов в имена на вкладке Outbounds.
// Таймер сохранён: стор групп событий не получает.
export const subscriptionGroupsStore = createPollingStore<SubscriptionGroup[]>(
	() => api.listSubscriptionGroups(),
	{ staleTime: 5_000, pollInterval: 30_000 },
);
