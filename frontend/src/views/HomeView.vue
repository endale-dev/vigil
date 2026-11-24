<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <main class="flex min-h-screen flex-col bg-slate-950 px-6 text-slate-100">
    <header class="mx-auto flex w-full max-w-6xl items-center justify-between py-5">
      <RouterLink class="text-lg font-semibold focus-visible:outline-2 focus-visible:outline-offset-4" :to="{ name: 'home' }">
        Vigil
      </RouterLink>
      <RouterLink
        class="rounded-md px-3 py-2 text-sm text-slate-200 hover:bg-slate-800 focus-visible:outline-2 focus-visible:outline-offset-2"
        :to="{ name: 'login' }"
      >
        Sign in
      </RouterLink>
    </header>

    <section class="mx-auto flex w-full max-w-6xl flex-1 flex-col justify-center py-16" aria-labelledby="home-title">
      <p class="text-sm font-medium uppercase tracking-wide text-emerald-300">Uptime monitoring</p>
      <h1 id="home-title" class="mt-4 max-w-2xl text-4xl font-semibold tracking-tight sm:text-5xl">
        Know when your services need attention.
      </h1>
      <p class="mt-5 max-w-xl text-lg leading-8 text-slate-300">
        Keep track of websites and network services from one simple dashboard.
      </p>
      <div class="mt-8 flex flex-wrap gap-3">
        <RouterLink
          class="min-h-11 rounded-md bg-emerald-700 px-5 py-3 font-medium text-white hover:bg-emerald-600 focus-visible:outline-2 focus-visible:outline-offset-2"
          :to="{ name: 'register' }"
        >
          Create an account
        </RouterLink>
        <RouterLink
          class="min-h-11 rounded-md border border-slate-600 px-5 py-3 font-medium hover:bg-slate-900 focus-visible:outline-2 focus-visible:outline-offset-2"
          :to="{ name: 'login' }"
        >
          Sign in
        </RouterLink>
      </div>
    </section>
  </main>
</template>
