import type {
	Event,
	Bot,
	Team,
	Competition,
	RankingBot,
	SearchResult,
	PaginatedResponse,
	EventFilters,
	BotFilters,
	AdminJob,
	AdminJobState
} from './types';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api';

export class APIError extends Error {
	status: number;
	body?: string;
	constructor(status: number, message: string, body?: string) {
		super(message);
		this.name = 'APIError';
		this.status = status;
		this.body = body;
	}
}

async function readErrorMessage(response: Response): Promise<string> {
	let text = '';
	try {
		text = await response.text();
	} catch {
		return response.statusText || 'Request failed';
	}
	if (!text) return response.statusText || 'Request failed';
	try {
		const asJson = JSON.parse(text) as { error?: string };
		if (asJson?.error) return asJson.error;
	} catch {
		// not json
	}
	return text;
}

async function fetchAPI<T>(endpoint: string): Promise<T> {
	const response = await fetch(`${API_BASE_URL}${endpoint}`);
	if (!response.ok) {
		const message = await readErrorMessage(response);
		throw new APIError(response.status, message, message);
	}
	return response.json();
}

async function postAPI<T>(endpoint: string, body: unknown): Promise<T> {
	const response = await fetch(`${API_BASE_URL}${endpoint}`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(body)
	});
	if (!response.ok) {
		const message = await readErrorMessage(response);
		throw new APIError(response.status, message, message);
	}
	return response.json();
}

export async function getEvents(
	page: number = 1,
	pageSize: number = 20,
	filters?: EventFilters
): Promise<PaginatedResponse<Event[]>> {
	const params = new URLSearchParams({
		page: page.toString(),
		page_size: pageSize.toString()
	});

	if (filters?.location) params.append('location', filters.location);
	if (filters?.start_date) params.append('start_date', filters.start_date);
	if (filters?.end_date) params.append('end_date', filters.end_date);
	if (filters?.weight_class) params.append('weight_class', filters.weight_class);
	if (filters?.sort) params.append('sort', filters.sort);

	return fetchAPI(`/events?${params}`);
}

export async function getEvent(id: string): Promise<Event> {
	return fetchAPI(`/events/${id}`);
}

export async function getCompetition(id: string): Promise<Competition> {
	return fetchAPI(`/competitions/${id}`);
}

export async function getBots(
	page: number = 1,
	pageSize: number = 20,
	filters?: BotFilters
): Promise<PaginatedResponse<Bot[]>> {
	const params = new URLSearchParams({
		page: page.toString(),
		page_size: pageSize.toString()
	});

	if (filters?.search) params.append('search', filters.search);
	if (filters?.weight_class) params.append('weight_class', filters.weight_class);
	if (filters?.team_id) params.append('team_id', filters.team_id);
	if (filters?.weapon) params.append('weapon', filters.weapon);
	if (filters?.year) params.append('year', filters.year);

	return fetchAPI(`/bots?${params}`);
}

export async function getBot(id: string): Promise<Bot> {
	return fetchAPI(`/bots/${id}`);
}

export async function getTeams(
	page: number = 1,
	pageSize: number = 20
): Promise<PaginatedResponse<Team[]>> {
	const params = new URLSearchParams({
		page: page.toString(),
		page_size: pageSize.toString()
	});

	return fetchAPI(`/teams?${params}`);
}

export async function getTeam(id: string): Promise<Team> {
	return fetchAPI(`/teams/${id}`);
}

export async function getRankings(year?: string, weightClass?: string): Promise<RankingBot[]> {
	const params = new URLSearchParams();
	if (year) params.append('year', year);
	if (weightClass) params.append('weight_class', weightClass);

	return fetchAPI(`/rankings?${params}`);
}

export async function getAvailableYears(): Promise<string[]> {
	return fetchAPI('/rankings/years');
}

export async function getAvailableWeightClasses(): Promise<string[]> {
	return fetchAPI('/rankings/weight-classes');
}

export async function search(query: string, limit: number = 10): Promise<SearchResult> {
	const params = new URLSearchParams({
		q: query,
		limit: limit.toString()
	});

	return fetchAPI(`/search?${params}`);
}

export async function adminStartJob(
	kind: string,
	options?: { year?: string; include_bots?: boolean }
): Promise<AdminJob> {
	return postAPI('/admin/jobs', { kind, ...(options || {}) });
}

export async function adminGetJob(id: string): Promise<AdminJob> {
	return fetchAPI(`/admin/jobs/${id}`);
}

export async function adminScrapeURL(url: string): Promise<AdminJob> {
	return postAPI('/admin/scrape-url', { url });
}

export async function adminListJobs(limit: number = 20): Promise<AdminJob[]> {
	const params = new URLSearchParams({ limit: String(limit) });
	return fetchAPI(`/admin/jobs?${params}`);
}

export async function adminCancelJob(id: string): Promise<AdminJob> {
	return postAPI(`/admin/jobs/${id}/cancel`, {});
}
