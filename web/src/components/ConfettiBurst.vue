<script setup lang="ts">
// Pluie de confettis décorative, jouée une fois au montage.
const tones = ['primary', 'secondary', 'success', 'info', 'error']
const pieces = Array.from({ length: 40 }, (_, i) => ({
  tone: tones[i % tones.length],
  left: Math.random() * 100,
  delay: Math.random() * 0.5,
  duration: 1.8 + Math.random() * 1.4,
  drift: (Math.random() - 0.5) * 220,
  spin: 360 + Math.random() * 720,
}))
</script>

<template>
  <div class="confetti" aria-hidden="true">
    <span
      v-for="(p, i) in pieces"
      :key="i"
      :style="{
        left: `${p.left}%`,
        background: `rgb(var(--v-theme-${p.tone}))`,
        animationDelay: `${p.delay}s`,
        animationDuration: `${p.duration}s`,
        '--drift': `${p.drift}px`,
        '--spin': `${p.spin}deg`,
      }"
    />
  </div>
</template>

<style scoped>
.confetti {
  position: fixed;
  inset: 0;
  z-index: 3000;
  overflow: hidden;
  pointer-events: none;
}

.confetti span {
  position: absolute;
  top: -20px;
  width: 8px;
  height: 14px;
  border-radius: 2px;
  animation: confetti-fall linear forwards;
}

@keyframes confetti-fall {
  to {
    transform: translate(var(--drift), 105vh) rotate(var(--spin));
  }
}

@media (prefers-reduced-motion: reduce) {
  .confetti {
    display: none;
  }
}
</style>
