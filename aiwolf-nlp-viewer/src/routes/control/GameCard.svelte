<script lang="ts">
  import { _ } from "svelte-i18n";

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

  let {
    game,
    onPause,
    onResume,
  }: {
    game: GameInfo;
    onPause?: () => void;
    onResume?: () => void;
  } = $props();
</script>

<div class="card bg-base-100 shadow-sm">
  <div class="card-body p-4">
    <div class="flex items-center justify-between flex-wrap gap-2">
      <div class="flex items-center gap-2">
        {#if game.finished}
          <iconify-icon
            inline
            icon="mdi:flag-checkered"
            class="text-lg opacity-50"
          ></iconify-icon>
        {:else if game.paused}
          <iconify-icon
            inline
            icon="mdi:pause-circle"
            class="text-lg text-warning"
          ></iconify-icon>
        {:else}
          <span class="badge badge-error badge-sm animate-pulse font-bold"
            >LIVE</span
          >
        {/if}
        <h2 class="card-title text-lg">
          {$_("control.game")}
          <span class="text-xs font-mono opacity-40"
            >{game.id.substring(0, 8)}</span
          >
        </h2>
      </div>
      <div class="flex items-center gap-2">
        <span class="badge badge-neutral">
          {game.day}{$_("control.dayUnit")}
          {game.is_daytime ? $_("control.daytime") : $_("control.night")}
        </span>
        {#if game.phase}
          <span class="badge badge-outline text-xs">{game.phase}</span>
        {/if}
        {#if game.finished && game.win_side}
          <span
            class="badge {game.win_side === 'VILLAGER'
              ? 'badge-success'
              : 'badge-error'}"
          >
            {game.win_side}
          </span>
        {/if}
        {#if !game.finished && onPause && onResume}
          {#if game.paused}
            <button class="btn btn-success btn-sm" onclick={onResume}>
              <iconify-icon inline icon="mdi:play"></iconify-icon>
              {$_("control.resume")}
            </button>
          {:else}
            <button class="btn btn-warning btn-sm" onclick={onPause}>
              <iconify-icon inline icon="mdi:pause"></iconify-icon>
              {$_("control.pause")}
            </button>
          {/if}
        {/if}
      </div>
    </div>

    <!-- エージェントグリッド -->
    {#if game.agents && game.agents.length > 0}
      <div class="flex flex-wrap gap-2 mt-3">
        {#each game.agents as agent}
          <div
            class="flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-sm
            {agent.has_error
              ? 'bg-error/10 text-error'
              : agent.alive
                ? 'bg-base-200'
                : 'bg-base-200 opacity-40 line-through'}"
          >
            <span class="font-mono font-bold">{agent.name}</span>
            <span class="text-xs opacity-50">{agent.team}</span>
            <span class="badge badge-xs badge-ghost">{agent.role}</span>
            {#if agent.has_error}
              <iconify-icon
                inline
                icon="mdi:alert-circle"
                class="text-error"
              ></iconify-icon>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>
