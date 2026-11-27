<script setup lang="ts">
// One sample per minute. A missing latency means the TCP connection timed out.
const responseTimes = [
  119, 108, 117, 146, 132, 126, 158, 142, 111, 104, 125, 120, 146, 173,
  152, 128, 135, 194, 168, 131, 122, 146, 149, 142, 135, 128, 121, null, null, null,
]
</script>

<template>
  <figure class="trace">
    <figcaption><span>Payments service</span><small>Sample TCP checks</small></figcaption>
    <div class="chart" role="img" aria-label="Twenty-seven successful connections followed by three timeouts, starting at 09:27 UTC.">
      <div class="scale" aria-hidden="true"><span>200 ms</span><span>100 ms</span><span>0 ms</span></div>
      <div class="plot" aria-hidden="true">
        <div v-for="(latency, index) in responseTimes" :key="index" class="interval" :class="{ timeout: latency === null }">
          <span v-if="latency !== null" class="bar" :style="{ height: latency / 2 + '%' }"></span>
        </div>
      </div>
      <div class="ticks" aria-hidden="true"><span>09:00</span><span>09:15</span><span>09:29</span></div>
    </div>
    <div class="key"><span><i></i>Response time</span><span><i class="failure"></i>Timeout</span><small>UTC</small></div>
  </figure>
</template>

<style scoped>
.trace { color: #eaf5e9; min-width: 0; }
figcaption, .key, .key > span, .ticks { display: flex; align-items: center; }
figcaption { justify-content: space-between; gap: 1rem; padding-bottom: 16px; border-bottom: 1px solid #517165; }
figcaption span { font-weight: 550; font-size: 17px; }
small, .ticks, .scale, .key { font: 11px var(--font-vigil-mono); }
small, .ticks, .scale { color: #b8d1c1; }
.chart { margin-block: 28px 24px; display: grid; grid-template-columns: 47px 1fr; gap: 9px; }
.scale { display: flex; flex-direction: column; justify-content: space-between; text-align: right; line-height: 1; }
.plot { height: clamp(120px, 12vw, 176px); display: flex; gap: clamp(2px, .28vw, 5px); align-items: end; border-block: 1px solid #4f6d60; background: linear-gradient(#436457,#436457) center / 100% 1px no-repeat; }
.interval { height: 100%; display: flex; flex: 1; align-items: end; }
.bar { width: 100%; background: repeating-linear-gradient(to top,#a2d9b2 0 5px,transparent 5px 7px); }
.timeout, .failure { background: repeating-linear-gradient(135deg,transparent 0 4px,#f5ac81 4px 6px); }
.ticks { grid-column: 2; justify-content: space-between; }
.key { gap: 17px; color: #dbe9df; }
.key > span { gap: 7px; }
.key i { width: 10px; height: 10px; }
.key i:first-child:not(.failure) { background: #a2d9b2; }
.key small { margin-left: auto; }
@media (max-width: 400px) {
  figcaption span { font-size: 15px; }
  .key, small, .ticks, .scale { font-size: 10px; }
  .chart { grid-template-columns: 40px 1fr; }
}
</style>
