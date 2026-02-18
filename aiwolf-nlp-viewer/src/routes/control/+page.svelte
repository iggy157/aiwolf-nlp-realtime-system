<script lang="ts">
  import { browser } from "$app/environment";
  import LanguageSwitcher from "$lib/components/LanguageSwitcher.svelte";
  import { onDestroy, onMount } from "svelte";
  import { _ } from "svelte-i18n";
  import "../../app.css";
  import GameCard from "./GameCard.svelte";

  // --- Types ---
  interface TeamInfo {
    name: string;
    count: number;
  }

  interface AgentInfo {
    idx: number;
    name: string;
    team: string;
    role: string;
    alive: boolean;
    has_error: boolean;
  }

  interface GameInfo {
    id: string;
    day: number;
    is_daytime: boolean;
    phase: string;
    paused: boolean;
    finished: boolean;
    win_side?: string;
    agents: AgentInfo[];
  }

  interface GameCostSummary {
    game_id: string;
    agents: CostReport[];
    total_cost: number;
    total_input_tokens: number;
    total_output_tokens: number;
  }

  interface CostReport {
    game_id: string;
    agent: string;
    team: string;
    model: string;
    llm_type: string;
    input_tokens: number;
    output_tokens: number;
    input_cost: number;
    output_cost: number;
    total_cost: number;
    call_count: number;
  }

  interface StatusResponse {
    server_version: string;
    manual_start: boolean;
    spawner_enabled: boolean;
    waiting_room: {
      required: number;
      teams: TeamInfo[];
      total: number;
    };
    games: GameInfo[];
    costs: GameCostSummary[];
    processes: ProcessInfo[];
  }

  interface ProcessInfo {
    id: string;
    team: string;
    count: number;
    model: string;
    llm_type: string;
    status: string;
  }

  // --- State ---
  let serverUrl = $state("http://localhost:8080");
  let connected = $state(false);
  let polling = $state(false);
  let status = $state<StatusResponse | null>(null);
  let error = $state<string | null>(null);
  let pollInterval: ReturnType<typeof setInterval> | null = null;

  // Spawn form state
  let spawnTeam = $state("team-a");
  let spawnCount = $state(5);
  let spawnLLMType = $state("google");
  let spawnModel = $state("gemini-2.0-flash-lite");
  let spawnTemp = $state(0.7);
  let spawning = $state(false);

  const defaultModels: Record<string, string> = {
    openai: "gpt-4o-mini",
    google: "gemini-2.0-flash-lite",
    ollama: "llama3.1",
  };

  // --- API Functions ---
  async function fetchStatus() {
    try {
      const resp = await fetch(`${serverUrl}/api/status`);
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
      status = await resp.json();
      connected = true;
      error = null;
    } catch (e) {
      connected = false;
      error = e instanceof Error ? e.message : "接続エラー";
    }
  }

  async function apiPost(path: string, body?: unknown) {
    try {
      const opts: RequestInit = { method: "POST" };
      if (body !== undefined) {
        opts.headers = { "Content-Type": "application/json" };
        opts.body = JSON.stringify(body);
      }
      const resp = await fetch(`${serverUrl}${path}`, opts);
      if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || `HTTP ${resp.status}`);
      }
      await fetchStatus();
    } catch (e) {
      error = e instanceof Error ? e.message : "APIエラー";
    }
  }

  function startPolling() {
    if (pollInterval) clearInterval(pollInterval);
    polling = true;
    fetchStatus();
    pollInterval = setInterval(fetchStatus, 2000);
  }

  function stopPolling() {
    if (pollInterval) {
      clearInterval(pollInterval);
      pollInterval = null;
    }
    polling = false;
    connected = false;
    status = null;
  }

  function startGame() {
    apiPost("/api/game/start");
  }

  function pauseGame(id: string) {
    apiPost(`/api/game/${id}/pause`);
  }

  function resumeGame(id: string) {
    apiPost(`/api/game/${id}/resume`);
  }

  async function spawnAgents() {
    spawning = true;
    try {
      await apiPost("/api/agent/spawn", {
        team: spawnTeam,
        count: spawnCount,
        llm_type: spawnLLMType,
        model: spawnModel,
        temperature: spawnTemp,
      });
    } finally {
      spawning = false;
    }
  }

  function stopProcess(id: string) {
    apiPost(`/api/agent/${id}/stop`);
  }

  // --- Derived ---
  const waitingReady = $derived(
    status
      ? status.waiting_room.total >= status.waiting_room.required
      : false,
  );

  const activeGames = $derived(
    status?.games.filter((g) => !g.finished) ?? [],
  );

  const finishedGames = $derived(
    status?.games.filter((g) => g.finished) ?? [],
  );

  const costs = $derived(status?.costs ?? []);

  const totalCostAllGames = $derived(
    costs.reduce((sum, c) => sum + c.total_cost, 0),
  );

  function formatCost(usd: number): string {
    if (usd === 0) return "$0";
    if (usd < 0.0001) return `$${usd.toFixed(6)}`;
    if (usd < 0.01) return `$${usd.toFixed(4)}`;
    return `$${usd.toFixed(2)}`;
  }

  function formatTokens(n: number): string {
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
    if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
    return String(n);
  }

  onMount(() => {
    if (browser) {
      const saved = localStorage.getItem("control_server_url");
      if (saved) serverUrl = saved;
    }
  });

  onDestroy(() => {
    if (pollInterval) clearInterval(pollInterval);
  });

  $effect(() => {
    if (browser) {
      localStorage.setItem("control_server_url", serverUrl);
    }
  });

  $effect(() => {
    spawnModel = defaultModels[spawnLLMType] || spawnModel;
  });
</script>

<main class="min-h-screen bg-base-300">
  <!-- Navbar -->
  <div class="navbar bg-base-100 shadow-sm">
    <a class="text-2xl font-bold ml-2" href="./">aiwolf-nlp-viewer</a>
    <span class="ml-2 text-sm opacity-50">/</span>
    <span class="ml-2 text-lg font-semibold">{$_("control.title")}</span>
    <div class="ml-auto flex items-center gap-2">
      <LanguageSwitcher />
      <label class="flex items-center cursor-pointer gap-2">
        <iconify-icon inline icon="mdi:white-balance-sunny"></iconify-icon>
        <input type="checkbox" value="dark" class="toggle theme-controller" />
        <iconify-icon inline icon="mdi:moon-and-stars"></iconify-icon>
      </label>
    </div>
  </div>

  <div class="max-w-5xl mx-auto p-4 space-y-4">
    <!-- 接続パネル -->
    <div class="card bg-base-100 shadow-sm">
      <div class="card-body p-4">
        <div class="flex items-center gap-3 flex-wrap">
          <div class="flex items-center gap-2">
            <div class="inline-grid *:[grid-area:1/1]">
              {#if connected}
                <div class="status status-success animate-ping"></div>
                <div class="status status-success"></div>
              {:else if polling}
                <div class="status status-warning animate-ping"></div>
                <div class="status status-warning"></div>
              {:else}
                <div class="status status-error"></div>
              {/if}
            </div>
            <span class="text-sm font-medium">
              {connected
                ? $_("control.connected")
                : polling
                  ? $_("control.connecting")
                  : $_("control.disconnected")}
            </span>
          </div>
          <label class="input flex-1 min-w-64">
            <iconify-icon class="opacity-50" inline icon="mdi:link"
            ></iconify-icon>
            <input
              type="text"
              class="grow"
              placeholder="http://localhost:8080"
              bind:value={serverUrl}
            />
          </label>
          {#if !polling}
            <button class="btn btn-primary" onclick={startPolling}>
              <iconify-icon inline icon="mdi:connection"></iconify-icon>
              {$_("control.connect")}
            </button>
          {:else}
            <button class="btn btn-error" onclick={stopPolling}>
              <iconify-icon inline icon="mdi:close-circle-outline"
              ></iconify-icon>
              {$_("control.disconnect")}
            </button>
          {/if}
        </div>
        {#if error}
          <div role="alert" class="alert alert-warning mt-2 py-2">
            <iconify-icon inline icon="mdi:alert-circle-outline"
            ></iconify-icon>
            <span class="text-sm">{error}</span>
          </div>
        {/if}
        {#if status?.server_version}
          <div class="text-xs opacity-40 mt-1">
            Server: v{status.server_version}
            {#if status.manual_start}
              <span class="badge badge-xs badge-primary ml-1"
                >Manual Start</span
              >
            {/if}
          </div>
        {/if}
      </div>
    </div>

    {#if connected && status}
      <!-- エージェント起動パネル -->
      {#if status.spawner_enabled}
        <div class="card bg-base-100 shadow-sm">
          <div class="card-body p-4">
            <h2 class="card-title text-lg">
              <iconify-icon inline icon="mdi:rocket-launch"></iconify-icon>
              {$_("control.spawnTitle")}
            </h2>
            <div class="grid grid-cols-2 sm:grid-cols-5 gap-2 mt-2">
              <label class="form-control">
                <div class="label py-0.5">
                  <span class="label-text text-xs"
                    >{$_("control.spawnTeam")}</span
                  >
                </div>
                <input
                  type="text"
                  class="input input-sm"
                  bind:value={spawnTeam}
                />
              </label>
              <label class="form-control">
                <div class="label py-0.5">
                  <span class="label-text text-xs"
                    >{$_("control.spawnCount")}</span
                  >
                </div>
                <input
                  type="number"
                  class="input input-sm"
                  bind:value={spawnCount}
                  min="1"
                  max="15"
                />
              </label>
              <label class="form-control">
                <div class="label py-0.5">
                  <span class="label-text text-xs"
                    >{$_("control.spawnLLM")}</span
                  >
                </div>
                <select class="select select-sm" bind:value={spawnLLMType}>
                  <option value="google">Google</option>
                  <option value="openai">OpenAI</option>
                  <option value="ollama">Ollama</option>
                </select>
              </label>
              <label class="form-control">
                <div class="label py-0.5">
                  <span class="label-text text-xs"
                    >{$_("control.spawnModel")}</span
                  >
                </div>
                <input
                  type="text"
                  class="input input-sm"
                  bind:value={spawnModel}
                />
              </label>
              <div class="flex items-end">
                <button
                  class="btn btn-primary btn-sm w-full"
                  onclick={spawnAgents}
                  disabled={spawning}
                >
                  {#if spawning}
                    <span class="loading loading-spinner loading-xs"></span>
                  {:else}
                    <iconify-icon inline icon="mdi:play"></iconify-icon>
                  {/if}
                  {$_("control.spawnStart")}
                </button>
              </div>
            </div>

            <!-- プロセス一覧 -->
            {#if status.processes.length > 0}
              <div class="flex flex-wrap gap-2 mt-3">
                {#each status.processes as proc}
                  <div
                    class="flex items-center gap-2 rounded-lg px-3 py-1.5 text-sm
                    {proc.status === 'running'
                      ? 'bg-success/10 text-success'
                      : proc.status === 'error'
                        ? 'bg-error/10 text-error'
                        : 'bg-base-200 opacity-50'}"
                  >
                    <span class="font-medium">{proc.team}</span>
                    <span class="text-xs opacity-60"
                      >×{proc.count} {proc.model}</span
                    >
                    <span class="badge badge-xs">{proc.status}</span>
                    {#if proc.status === "running"}
                      <button
                        class="btn btn-ghost btn-xs"
                        onclick={() => stopProcess(proc.id)}
                      >
                        <iconify-icon inline icon="mdi:stop"></iconify-icon>
                      </button>
                    {/if}
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        </div>
      {/if}

      <!-- 待機部屋 -->
      <div class="card bg-base-100 shadow-sm">
        <div class="card-body p-4">
          <div class="flex items-center justify-between flex-wrap gap-2">
            <h2 class="card-title text-lg">
              <iconify-icon inline icon="mdi:account-group"></iconify-icon>
              {$_("control.waitingRoom")}
            </h2>
            <div class="flex items-center gap-2">
              <span
                class="badge {waitingReady
                  ? 'badge-success'
                  : 'badge-neutral'} badge-lg font-mono"
              >
                {status.waiting_room.total} / {status.waiting_room.required}
              </span>
              {#if waitingReady}
                <button class="btn btn-success btn-sm" onclick={startGame}>
                  <iconify-icon inline icon="mdi:play"></iconify-icon>
                  {$_("control.startGame")}
                </button>
              {/if}
            </div>
          </div>

          {#if status.waiting_room.teams.length > 0}
            <div class="flex flex-wrap gap-2 mt-2">
              {#each status.waiting_room.teams as team}
                <div class="badge badge-outline gap-1">
                  <iconify-icon inline icon="mdi:account"></iconify-icon>
                  {team.name}
                  {#if team.count > 1}
                    <span class="badge badge-sm badge-neutral"
                      >×{team.count}</span
                    >
                  {/if}
                </div>
              {/each}
            </div>
          {:else}
            <p class="text-sm opacity-50 mt-2">
              {$_("control.noTeamsWaiting")}
            </p>
          {/if}
        </div>
      </div>

      <!-- アクティブゲーム -->
      {#each activeGames as game}
        <GameCard
          {game}
          onPause={() => pauseGame(game.id)}
          onResume={() => resumeGame(game.id)}
        />
      {/each}

      <!-- コスト集計 -->
      {#if costs.length > 0}
        <div class="card bg-base-100 shadow-sm">
          <div class="card-body p-4">
            <div class="flex items-center justify-between">
              <h2 class="card-title text-lg">
                <iconify-icon inline icon="mdi:currency-usd"></iconify-icon>
                {$_("control.costTracking")}
              </h2>
              <span class="badge badge-lg badge-accent font-mono">
                {$_("control.totalCost")}: {formatCost(totalCostAllGames)}
              </span>
            </div>

            {#each costs as cost}
              <div class="mt-3">
                <div class="flex items-center gap-2 mb-2">
                  <span class="text-xs font-mono opacity-50"
                    >{cost.game_id.substring(0, 8)}</span
                  >
                  <span class="badge badge-neutral font-mono"
                    >{formatCost(cost.total_cost)}</span
                  >
                  <span class="text-xs opacity-40">
                    ↑{formatTokens(cost.total_input_tokens)} ↓{formatTokens(
                      cost.total_output_tokens,
                    )}
                  </span>
                </div>
                <div class="overflow-x-auto">
                  <table class="table table-xs">
                    <thead>
                      <tr>
                        <th>{$_("control.costAgent")}</th>
                        <th>{$_("control.costTeam")}</th>
                        <th>{$_("control.costModel")}</th>
                        <th class="text-right"
                          >{$_("control.costInputTokens")}</th
                        >
                        <th class="text-right"
                          >{$_("control.costOutputTokens")}</th
                        >
                        <th class="text-right">{$_("control.costCalls")}</th>
                        <th class="text-right">{$_("control.costTotal")}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {#each cost.agents as agent}
                        <tr>
                          <td class="font-mono">{agent.agent}</td>
                          <td>{agent.team}</td>
                          <td class="text-xs opacity-60">{agent.model}</td>
                          <td class="text-right font-mono"
                            >{formatTokens(agent.input_tokens)}</td
                          >
                          <td class="text-right font-mono"
                            >{formatTokens(agent.output_tokens)}</td
                          >
                          <td class="text-right font-mono"
                            >{agent.call_count}</td
                          >
                          <td class="text-right font-mono font-bold"
                            >{formatCost(agent.total_cost)}</td
                          >
                        </tr>
                      {/each}
                    </tbody>
                  </table>
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/if}

      <!-- 終了済みゲーム -->
      {#if finishedGames.length > 0}
        <div class="collapse collapse-arrow bg-base-100 shadow-sm">
          <input type="checkbox" />
          <div class="collapse-title text-lg font-medium">
            <iconify-icon inline icon="mdi:history"></iconify-icon>
            {$_("control.finishedGames")} ({finishedGames.length})
          </div>
          <div class="collapse-content space-y-2">
            {#each finishedGames as game}
              <GameCard {game} />
            {/each}
          </div>
        </div>
      {/if}

      <!-- 何もない場合 -->
      {#if activeGames.length === 0 && finishedGames.length === 0 && status.waiting_room.teams.length === 0}
        <div class="text-center py-12 opacity-40">
          <iconify-icon
            icon="mdi:gamepad-variant-outline"
            style="font-size: 4rem"
          ></iconify-icon>
          <p class="mt-2">{$_("control.noActivity")}</p>
        </div>
      {/if}
    {/if}
  </div>
</main>
