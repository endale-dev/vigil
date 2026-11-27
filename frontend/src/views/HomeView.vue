<script setup lang="ts">
import { ArrowDown, ArrowRight, ArrowUp, Globe2, Mail, Send } from '@lucide/vue'
import { RouterLink } from 'vue-router'
import LandingCheckTrace from '../components/LandingCheckTrace.vue'
import VigilBrand from '../components/VigilBrand.vue'
import '../console.css'
</script>

<template>
  <div id="top" class="landing console-theme">
    <a class="skip-link" href="#main">Skip to content</a>
    <header class="masthead wrap">
      <a href="#top" aria-label="Vigil home"><VigilBrand /></a>
      <nav aria-label="Explore Vigil">
        <a href="#workspace">Workspace</a><a href="#incidents">Incidents</a><a href="#alerts">Alerts</a>
      </nav>
      <RouterLink class="signin" :to="{ name: 'login' }">Sign in <ArrowRight :size="15" aria-hidden="true" /></RouterLink>
    </header>

    <main id="main">
      <section class="hero" aria-labelledby="hero-title">
        <div class="hero-inner wrap">
          <div>
            <h1 id="hero-title">Is your service <em>responding?</em></h1>
            <p>Vigil checks your websites, ports, and certificates and lets you know when something stops working.</p>
            <div class="actions">
              <RouterLink class="button" :to="{ name: 'register' }">Create account <ArrowRight :size="17" aria-hidden="true" /></RouterLink>
              <a class="tour-link" href="#workspace">See the workspace <ArrowDown :size="15" aria-hidden="true" /></a>
            </div>
          </div>
          <LandingCheckTrace />
        </div>
      </section>

      <section id="workspace" class="workspace-preview wrap" aria-labelledby="workspace-title">
        <div class="section-intro">
          <h2 id="workspace-title">One view of your services.</h2>
          <p>Find the monitor that needs attention. Each row shows its current status, its latest check, and its uptime over the past 24 hours.</p>
        </div>
        <figure>
          <div class="screen-mat">
            <picture>
              <source media="(max-width: 620px)" srcset="/images/vigil-mobile-overview.png" width="531" height="737" />
              <img src="/images/vigil-dashboard-preview.png" width="1440" height="700" alt="Vigil overview with four active monitors. Three are up and Payments service is down." />
            </picture>
          </div>
          <figcaption>Dashboard overview <span>Sample workspace</span></figcaption>
        </figure>
        <dl class="protocols">
          <div><dt>HTTP</dt><dd>Check a website or API for the status code you expect.</dd></div>
          <div><dt>TCP</dt><dd>Test whether a host accepts a connection on a specific port.</dd></div>
          <div><dt>TLS</dt><dd>Check certificate validity and set an expiry warning window.</dd></div>
        </dl>
      </section>

      <section id="incidents" class="incident-section" aria-labelledby="incident-title">
        <div class="incident-inner wrap">
          <div class="incident-copy">
            <p class="section-label">Incident history</p>
            <h2 id="incident-title">At 09:27, payments stopped responding.</h2>
            <p>The last successful connection took 121 ms. The next three timed out. Open a monitor to see the individual results and when the incident began.</p>
          </div>
          <figure>
            <img src="/images/vigil-check-history.png" width="870" height="716" loading="lazy" alt="Recent checks for Payments service. The checks at 09:27, 09:28, and 09:29 each timed out after five seconds." />
            <figcaption>Payments service <span>Sample check history · UTC</span></figcaption>
          </figure>
        </div>
      </section>

      <section id="alerts" class="alerts wrap" aria-labelledby="alerts-title">
        <div>
          <p class="section-label">Alerts &amp; public status</p>
          <h2 id="alerts-title">Who needs<br /> to know?</h2>
        </div>
        <dl class="channels">
          <div><dt><Mail aria-hidden="true" /> Email</dt><dd>Send outage and recovery notices to the inbox you use for alerts.</dd></div>
          <div><dt><Send aria-hidden="true" /> Telegram</dt><dd>Connect the Vigil bot to a personal chat or group to receive the same updates in Telegram.</dd></div>
          <div><dt><Globe2 aria-hidden="true" /> Public status page</dt><dd>Choose the services your users can see and share their current status with one link.</dd></div>
        </dl>
      </section>
    </main>

    <footer class="footer wrap">
      <a href="#top" aria-label="Vigil home"><VigilBrand /></a>
      <a class="back" href="#top">Back to top <ArrowUp :size="15" aria-hidden="true" /></a>
    </footer>
  </div>
</template>

<style scoped>
.landing {
  --ink: #192f27;
  --subtle: #51645a;
  --rule: #cdd8ce;
  color: var(--ink);
  background: #fbfcf8;
}
.wrap { width: min(1180px, calc(100% - clamp(40px, 9vw, 144px))); margin-inline: auto; }
.skip-link { position: absolute; left: 12px; top: 10px; padding: 10px; background: white; z-index: 2; transform: translateY(-150%); }
.skip-link:focus { transform: none; }
.masthead, .masthead nav, .signin, .actions, .button, .tour-link, .footer, .back { display: flex; align-items: center; }
.masthead { min-height: 78px; gap: 40px; }
.masthead nav { margin-left: auto; gap: 29px; }
.masthead a, .back { font-size: 14px; font-weight: 550; }
.masthead a:hover, .footer a:hover { color: #17624d; }
.signin { gap: 10px; border-left: 1px solid var(--rule); padding: 4px 0 4px 26px; }
.hero { background: #113c30; color: #f7f9ee; }
.hero-inner { display: grid; grid-template-columns: 1fr .94fr; gap: clamp(36px, 6vw, 100px); align-items: center; padding-block: clamp(64px, 7.5vw, 114px); }
h1, h2 { text-wrap: balance; }
h1 { font-size: clamp(58px, 5.9vw, 90px); line-height: .99; letter-spacing: -.062em; font-weight: 500; }
h1 em { display: block; color: #c1e8c3; font-family: Georgia, serif; font-weight: 400; letter-spacing: -.065em; }
.hero-inner > div > p { max-width: 40ch; margin-top: 24px; color: #d7e6db; font-size: clamp(16px, 1.45vw, 20px); line-height: 1.52; }
.actions { gap: 24px; flex-wrap: wrap; margin-top: 30px; }
.button, .tour-link, .back { gap: 9px; }
.button { min-height: 45px; background: #d3efd4; color: #133e32; padding: 10px 17px; border-radius: 4px; font-weight: 650; }
.button:hover { background: #e3f6df; }
.tour-link { font-size: 14px; padding-block: 9px; }
.tour-link:hover, .back:hover, .masthead nav a:hover { text-decoration: underline; text-underline-offset: 4px; }
.hero a:focus-visible { outline-color: #d3efd4; }
h2 { font-weight: 520; font-size: clamp(36px, 4vw, 60px); line-height: 1.04; letter-spacing: -.056em; }
.workspace-preview, .incident-inner, .alerts { padding-block: clamp(56px, 6.8vw, 102px); }
.section-intro, .incident-inner, .alerts { display: grid; gap: clamp(28px, 6vw, 90px); }
.section-intro { grid-template-columns: 1.2fr 1fr; align-items: end; margin-bottom: 32px; }
.section-intro > p, .incident-copy > p:last-child { color: var(--subtle); font-size: clamp(16px, 1.38vw, 19px); line-height: 1.56; max-width: 46ch; }
.screen-mat { padding: clamp(16px, 3.5vw, 55px); background: #e3ece3; border: 1px solid #cfdbd0; }
figure img { width: 100%; height: auto; display: block; }
.screen-mat img { border: 1px solid #bfcfc3; box-shadow: 0 6px 18px #1a39240b; }
figure > figcaption { display: flex; align-items: baseline; flex-wrap: wrap; justify-content: space-between; gap: 4px 12px; padding-block: 11px; color: var(--subtle); font: 11px/1.5 var(--font-vigil-mono); }
.protocols { display: grid; grid-template-columns: repeat(3, 1fr); margin-top: 25px; border-top: 1px solid var(--rule); }
.protocols > div { padding: 22px 26px 0 0; }
.protocols dt, .section-label { font: 12px/1.4 var(--font-vigil-mono); text-transform: uppercase; letter-spacing: .08em; color: #216448; }
.protocols dt { font-size: 16px; }
.protocols dd { max-width: 31ch; margin-top: 8px; color: var(--subtle); line-height: 1.52; }
.incident-section { background: #edf1e9; border-block: 1px solid var(--rule); }
.incident-inner { grid-template-columns: .95fr 1.05fr; align-items: center; }
.incident-inner figure { max-width: 550px; }
.incident-copy h2, .alerts h2 { margin-block: 16px 20px; }
.incident-copy h2 { max-width: 14ch; }
.alerts { grid-template-columns: .8fr 1.2fr; }
.channels > div { display: grid; grid-template-columns: minmax(150px, .8fr) 1.35fr; gap: 18px; padding-block: 22px; border-top: 1px solid var(--rule); }
.channels dt { display: flex; align-items: baseline; gap: 10px; font-size: 17px; font-weight: 600; }
.channels svg { width: 18px; height: 18px; flex-shrink: 0; align-self: flex-start; margin-top: 4px; color: #17624d; }
.channels dd { color: var(--subtle); line-height: 1.55; }
.footer { border-top: 1px solid var(--rule); padding-block: 24px; justify-content: space-between; }
@media (max-width: 820px) {
  .hero-inner { grid-template-columns: 1fr 1fr; gap: 28px; }
  h1 { font-size: clamp(49px, 7.3vw, 62px); }
  .actions { gap: 12px; }
  .section-intro, .incident-inner, .alerts { gap: 28px; }
  .channels > div { display: block; }
  .channels dd { margin-top: 8px; }
}
@media (max-width: 650px) {
  .masthead { min-height: 0; flex-wrap: wrap; gap: 0; justify-content: space-between; padding-top: 14px; }
  .masthead nav { order: 1; width: 100%; gap: 24px; margin-top: 12px; padding-block: 10px; border-top: 1px solid var(--rule); }
  .signin { border: 0; padding: 8px 0; }
  .hero-inner, .section-intro, .incident-inner, .alerts { grid-template-columns: 1fr; }
  .hero-inner { gap: 42px; padding-block: 46px; }
  h1 { font-size: clamp(55px, 11vw, 72px); }
  .hero-inner > div > p { margin-top: 18px; }
  .actions { gap: 19px; margin-top: 23px; }
  .section-intro { gap: 14px; margin-bottom: 20px; }
  .protocols { grid-template-columns: 1fr; margin-top: 12px; }
  .protocols > div { display: grid; grid-template-columns: 52px 1fr; gap: 12px; padding: 15px 0; border-bottom: 1px solid var(--rule); }
  .protocols dd { max-width: none; margin: 0; }
  .incident-copy h2 { max-width: 17ch; }
  .alerts h2 { margin-bottom: 0; }
  .alerts h2 br { display: none; }
  .channels > div { padding-block: 16px; }
}
@media (max-width: 620px) {
  .screen-mat { padding: 16px; }
  .screen-mat picture { display: block; max-width: 354px; margin-inline: auto; }
}
</style>
