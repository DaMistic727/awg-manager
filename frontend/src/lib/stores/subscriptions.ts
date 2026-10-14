import { api } from '$lib/api/client';
import { createPollingStore } from './polling';
import { registerStore } from './storeRegistry';
import type { Subscription, SubscriptionGroup } from '$lib/types';

// Подписки обновляет событие: бэкенд публикует singbox.subscriptions на
// каждую запись стора — REST, MCP и фоновое обновление — и на тумблер NDMS
// Proxy (MC9). Серия записей одного обновления не теряется: invalidate во
// время запроса даёт повтор после него (F663).
export const subscriptionsStore = createPollingStore<Subscription[]>(
	() => api.listSubscriptions(),
	{ staleTime: 5_000, pollInterval: 0 },
);
registerStore('singbox.subscriptions', subscriptionsStore);

// #572: сводные группы для резолва agg-тегов в имена на вкладке Outbounds.
// Таймер сохранён: стор групп событий не получает.
export const subscriptionGroupsStore = createPollingStore<SubscriptionGroup[]>(
	() => api.listSubscriptionGroups(),
	{ staleTime: 5_000, pollInterval: 30_000 },
);
