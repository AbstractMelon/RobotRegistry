export interface Event {
	id: string;
	name: string;
	url: string;
	start_date: string;
	end_date: string;
	location: string;
	latitude?: string;
	longitude?: string;
	bots_count: number;
	logo_url: string;
	description?: string;
	website?: string;
	organizer?: string;
	competitions?: Competition[];
}

export interface Competition {
	id: string;
	event_id: string;
	name: string;
	weight_class: string;
	url: string;
	date: string;
	begin_time: string;
	end_time: string;
	location: string;
	max_combatants: number;
	min_combatants: number;
	max_age: string;
	min_age: string;
	registration_fee: string;
	participants?: Participant[];
}

export interface Participant {
	bot_name: string;
	bot_id: string;
	bot_url: string;
	team_name: string;
	team_id: string;
	team_url: string;
	status: string;
	bot_image: string;
}

export interface Bot {
	id: string;
	name: string;
	url: string;
	rank: number;
	weight_class: string;
	points: number;
	team: string;
	team_id: string;
	team_url: string;
	weapons: string[];
	description: string;
	image_url: string;
	years: string[];
	history?: BotHistory[];
	rankings?: BotRanking[];
}

export interface BotRanking {
	year: string;
	weight_class: string;
	rank: number;
	points: number;
}

export interface BotHistory {
	event_name: string;
	event_url: string;
	competition_url: string;
	place: string;
	points: number;
}

export interface Team {
	id: string;
	name: string;
	url: string;
	logo_url: string;
	bot_ids: string[];
	bot_names: string[];
	bot_urls: string[];
}

export interface RankingBot {
	id: string;
	name: string;
	url: string;
	rank: number;
	weight_class: string;
	points: number;
	team: string;
	team_id: string;
	team_url: string;
	image_url: string;
}

export interface SearchResult {
	events: Event[];
	bots: Bot[];
	teams: Team[];
}

export interface PaginatedResponse<T> {
	data: T;
	page: number;
	page_size: number;
	total_items: number;
	total_pages: number;
}

export interface WeightClass {
	name: string;
	count: number;
}

export interface Year {
	year: number;
	count: number;
}

export interface RankingsResponse {
	[key: string]: RankingBot[];
}

export interface EventFilters {
	location?: string;
	start_date?: string;
	end_date?: string;
	weight_class?: string;
	sort?: 'start_date_asc' | 'start_date_desc';
}

export interface BotFilters {
	search?: string;
	weight_class?: string;
	team_id?: string;
	weapon?: string;
	year?: string;
}
