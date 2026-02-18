<script lang="ts">
  import { base } from "$app/paths";
  import { getAgentAvatar, getAgentColor, type Talk } from "$lib/types/agent";

  let {
    talk,
    agents,
    showHeader = true,
  }: { talk: Talk; agents: string[]; showHeader?: boolean } = $props();

  const avatar = $derived(`${base}${getAgentAvatar(talk.agent)}`);
  const color = $derived(getAgentColor(talk.agent, agents));
</script>

{#if talk.over || talk.skip}
  <!-- Over/Skip はシステムメッセージとして表示 -->
  <div class="flex items-center justify-center gap-2 py-1">
    <div
      class="flex items-center gap-1.5 rounded-full px-3 py-0.5 text-xs opacity-50"
      style="background: {color.bg};"
    >
      {#if talk.over}
        <iconify-icon inline icon="mdi:skip-forward" style="font-size:0.85em"
        ></iconify-icon>
      {:else}
        <iconify-icon
          inline
          icon="mdi:arrow-u-down-right-bold"
          style="font-size:0.85em"
        ></iconify-icon>
      {/if}
      <span class="font-medium" style="color: {color.text};">{talk.agent}</span>
    </div>
  </div>
{:else}
  <div class="flex items-end gap-2.5 {showHeader ? 'mt-3' : 'mt-0.5'}">
    <!-- アバター（グループの先頭のみ表示） -->
    <div class="flex-shrink-0 w-9">
      {#if showHeader}
        <div
          class="w-9 h-9 rounded-full overflow-hidden ring-2 ring-base-300 shadow-sm"
        >
          <img src={avatar} alt={talk.agent} class="w-full h-full object-cover" />
        </div>
      {/if}
    </div>

    <div class="flex flex-col max-w-[75%] min-w-0">
      <!-- エージェント名（グループの先頭のみ） -->
      {#if showHeader}
        <span
          class="text-xs font-semibold mb-0.5 ml-1"
          style="color: {color.text};">{talk.agent}</span
        >
      {/if}

      <!-- メッセージバブル -->
      <div
        class="rounded-2xl rounded-bl-md px-3.5 py-2 shadow-sm text-sm leading-relaxed break-all whitespace-pre-wrap"
        style="background: {color.bg}; color: {color.text};"
      >
        {talk.text}
      </div>
    </div>
  </div>
{/if}