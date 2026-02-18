<script lang="ts">
  import type { Talk } from "$lib/types/agent";
  import { tick } from "svelte";
  import { _ } from "svelte-i18n";
  import ChatBubble from "./ChatBubble.svelte";

  let {
    header,
    talks,
    agents,
    isLive = false,
  }: {
    header: string;
    talks: Talk[];
    agents: string[];
    isLive?: boolean;
  } = $props();

  let scrollContainer: HTMLDivElement | undefined = $state(undefined);
  let shouldAutoScroll = $state(true);
  let prevTalkCount = $state(0);

  // スクロール位置のチェック（ユーザが上にスクロールした場合は自動スクロールを停止）
  function onScroll() {
    if (!scrollContainer) return;
    const { scrollTop, scrollHeight, clientHeight } = scrollContainer;
    shouldAutoScroll = scrollHeight - scrollTop - clientHeight < 80;
  }

  // 新しいメッセージが来たら自動スクロール
  $effect(() => {
    if (talks.length > prevTalkCount && shouldAutoScroll && scrollContainer) {
      prevTalkCount = talks.length;
      tick().then(() => {
        scrollContainer?.scrollTo({
          top: scrollContainer.scrollHeight,
          behavior: "smooth",
        });
      });
    } else {
      prevTalkCount = talks.length;
    }
  });
</script>

<div class="flex-1 min-w-[360px] max-w-[640px] rounded-xl bg-base-200 flex flex-col h-full shadow-sm">
  <!-- ヘッダー -->
  <div
    class="flex items-center gap-2 px-4 py-2.5 border-b border-base-300 flex-shrink-0"
  >
    <h2 class="text-base font-bold flex-1">{header}</h2>
    {#if isLive}
      <span
        class="flex items-center gap-1.5 text-xs font-semibold text-success"
      >
        <span class="relative flex h-2 w-2">
          <span
            class="animate-ping absolute inline-flex h-full w-full rounded-full bg-success opacity-75"
          ></span>
          <span class="relative inline-flex rounded-full h-2 w-2 bg-success"
          ></span>
        </span>
        LIVE
      </span>
    {/if}
    <span class="text-xs opacity-50">{$_("agent.messageCount", { values: { count: talks.length } })}</span>
  </div>

  <!-- チャットエリア -->
  <div
    bind:this={scrollContainer}
    onscroll={onScroll}
    class="flex-1 overflow-y-auto px-3 py-2"
  >
    {#if talks.length === 0}
      <div class="flex items-center justify-center h-full opacity-40">
        <p class="text-sm">{$_("agent.noMessages")}</p>
      </div>
    {:else}
      {#each talks as talk, i}
        <!-- 日付区切り -->
        {#if i === 0 || talk.day !== talks[i - 1].day}
          <div class="flex items-center gap-3 my-3">
            <div class="flex-1 h-px bg-base-300"></div>
            <span class="text-xs font-semibold opacity-50 whitespace-nowrap"
              >{$_("agent.dayLabel", { values: { day: talk.day } })}</span
            >
            <div class="flex-1 h-px bg-base-300"></div>
          </div>
        {/if}

        <!-- メッセージ（同一エージェントの連続発言はヘッダーを省略） -->
        <ChatBubble
          {talk}
          {agents}
          showHeader={i === 0 ||
            talks[i - 1].agent !== talk.agent ||
            talks[i - 1].day !== talk.day ||
            talks[i - 1].over ||
            talks[i - 1].skip}
        />
      {/each}
    {/if}
  </div>
</div>
