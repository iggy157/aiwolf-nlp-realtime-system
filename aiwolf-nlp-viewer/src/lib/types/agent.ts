import { Species, type Role, type Status } from "$lib/constants/common";

export interface Packet {
    request: Request;
    info: Info | undefined;
    setting: Setting | undefined;
    talk_history: Talk[] | undefined;
    whisper_history: Talk[] | undefined;
}

export enum Request {
    NAME = "NAME",
    TALK = "TALK",
    WHISPER = "WHISPER",
    VOTE = "VOTE",
    DIVINE = "DIVINE",
    GUARD = "GUARD",
    ATTACK = "ATTACK",
    INITIALIZE = "INITIALIZE",
    DAILY_INITIALIZE = "DAILY_INITIALIZE",
    DAILY_FINISH = "DAILY_FINISH",
    FINISH = "FINISH",
    TALK_START = "TALK_START",
    TALK_BROADCAST = "TALK_BROADCAST",
    TALK_END = "TALK_END",
    WHISPER_START = "WHISPER_START",
    WHISPER_BROADCAST = "WHISPER_BROADCAST",
    WHISPER_END = "WHISPER_END"
}

export const REALTIME_REQUESTS = new Set([
    Request.TALK_START, Request.TALK_BROADCAST, Request.TALK_END,
    Request.WHISPER_START, Request.WHISPER_BROADCAST, Request.WHISPER_END,
]);

export function isRealtimeRequest(request: Request): boolean {
    return REALTIME_REQUESTS.has(request);
}

export interface Info {
    game_id: string;
    day: number;
    agent: string;
    profile: string | undefined;
    medium_result: Judge | undefined;
    divine_result: Judge | undefined;
    executed_agent: string | undefined;
    attacked_agent: string | undefined;
    vote_list: Vote[] | undefined;
    attack_vote_list: Vote[] | undefined;
    status_map: Record<string, Status>;
    role_map: Record<string, Role>;
    remain_count: number | undefined;
    remain_length: number | undefined;
    remain_skip: number | undefined;
}

export interface Judge {
    day: number;
    agent: string;
    target: string;
    result: Species;
}

export interface Vote {
    day: number;
    agent: string;
    target: string;
}

export interface MaxLength {
    count_in_word?: boolean;
    per_talk?: number;
    mention_length?: number;
    per_agent?: number;
    base_length?: number;
}

export interface MaxCount {
    per_agent: number;
    per_day: number;
}

export interface TalkSettings {
    max_count: MaxCount;
    max_length: MaxLength;
    max_skip: number;
}

export interface VoteSettings {
    max_count: number;
}

export interface AttackVoteSettings {
    max_count: number;
    allow_no_target: boolean;
}

export interface TimeoutSettings {
    action: number;
    response: number;
}

export interface Setting {
    agent_count: number;
    role_num_map: Record<Role, number>;
    vote_visibility: boolean;
    talk_on_first_day: boolean;
    talk: TalkSettings;
    whisper: TalkSettings;
    vote: VoteSettings;
    attack_vote: AttackVoteSettings;
    timeout: TimeoutSettings;
}

export interface Talk {
    idx: number;
    day: number;
    turn: number;
    agent: string;
    text: string;
    skip: boolean;
    over: boolean;
}

export const DefaultProfileAvatars: Record<string, string> = {
    'ミナト': '/images/male/01.png',
    'タクミ': '/images/male/02.png',
    'ケンジ': '/images/male/03.png',
    'リュウジ': '/images/male/04.png',
    'ダイスケ': '/images/male/05.png',
    'シオン': '/images/male/06.png',
    'ベンジャミン': '/images/male/07.png',
    'トシオ': '/images/male/08.png',
    'ジョナサン': '/images/male/09.png',
    'シュンイチ': '/images/male/10.png',
    'ジョージ': '/images/male/11.png',
    'セルヴァス': '/images/male/12.png',
    'サクラ': '/images/female/01.png',
    'リン': '/images/female/02.png',
    'ユミ': '/images/female/03.png',
    'メイ': '/images/female/04.png',
    'ミサキ': '/images/female/05.png',
    'ミオ': '/images/female/06.png',
    'ミヅキ': '/images/female/07.png',
    'ミナコ': '/images/female/08.png',
    'アスカ': '/images/female/09.png',
    'ミドリ': '/images/female/10.png',
    'ヴィクトリア': '/images/female/11.png',
    'シズエ': '/images/female/12.png'
}

// Agent[XX] 形式の名前からアバター画像パスを取得
// 日本語名の場合は DefaultProfileAvatars から、Agent[XX] は番号ベースで割り当て
const agentNumberAvatars: string[] = [
    '/images/male/01.png', '/images/female/01.png',
    '/images/male/02.png', '/images/female/02.png',
    '/images/male/03.png', '/images/female/03.png',
    '/images/male/04.png', '/images/female/04.png',
    '/images/male/05.png', '/images/female/05.png',
    '/images/male/06.png', '/images/female/06.png',
    '/images/male/07.png', '/images/female/07.png',
    '/images/male/08.png', '/images/female/08.png',
    '/images/male/09.png', '/images/female/09.png',
    '/images/male/10.png', '/images/female/10.png',
    '/images/male/11.png', '/images/female/11.png',
    '/images/male/12.png', '/images/female/12.png',
];

export function getAgentAvatar(agentName: string): string {
    // 日本語名の場合
    if (DefaultProfileAvatars[agentName]) {
        return DefaultProfileAvatars[agentName];
    }
    // Agent[XX] 形式の場合
    const match = agentName.match(/\[(\d+)\]/);
    if (match) {
        const idx = parseInt(match[1], 10) - 1;
        return agentNumberAvatars[idx % agentNumberAvatars.length];
    }
    // フォールバック
    return agentNumberAvatars[0];
}

// エージェントごとのバブル色パレット（DaisyUI/Tailwind互換のHSLカラー）
const AGENT_COLORS = [
    { bg: '#dbeafe', text: '#1e3a5f' },  // blue
    { bg: '#fce7f3', text: '#701a45' },  // pink
    { bg: '#d1fae5', text: '#064e3b' },  // green
    { bg: '#fef3c7', text: '#78350f' },  // amber
    { bg: '#e0e7ff', text: '#312e81' },  // indigo
    { bg: '#ffe4e6', text: '#881337' },  // rose
    { bg: '#ccfbf1', text: '#134e4a' },  // teal
    { bg: '#fde68a', text: '#713f12' },  // yellow
    { bg: '#e9d5ff', text: '#581c87' },  // purple
    { bg: '#fed7aa', text: '#7c2d12' },  // orange
    { bg: '#cffafe', text: '#155e75' },  // cyan
    { bg: '#f5d0fe', text: '#701a75' },  // fuchsia
    { bg: '#d9f99d', text: '#365314' },  // lime
];

export function getAgentColor(agentName: string, agents: string[]): { bg: string; text: string } {
    const idx = agents.indexOf(agentName);
    if (idx >= 0) {
        return AGENT_COLORS[idx % AGENT_COLORS.length];
    }
    // Agent[XX]から番号を取得
    const match = agentName.match(/\[(\d+)\]/);
    if (match) {
        const num = parseInt(match[1], 10) - 1;
        return AGENT_COLORS[num % AGENT_COLORS.length];
    }
    return AGENT_COLORS[0];
}
