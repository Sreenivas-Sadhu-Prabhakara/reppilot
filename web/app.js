/* RepPilot front-end — vanilla JS, no build step. */
"use strict";

const state = {
  profile: null,
  tracked: 0,
  unanswered: 0,
  reviews: [],
  counts: { total: 0, unanswered: 0, answered: 0 },
  campaigns: [],
  outbox: [],
  digest: null,
  filter: "",
  starFilter: "",
  tone: "Professional",
  language: "English",
  openDrafts: new Set(), // review ids with an open draft editor
};

const $ = (sel, root) => (root || document).querySelector(sel);
const $$ = (sel, root) => Array.from((root || document).querySelectorAll(sel));

/* ---------- api ---------- */

async function api(path, opts = {}) {
  const res = await fetch("/api/v1" + path, {
    headers: { "Content-Type": "application/json" },
    ...opts,
  });
  let body = null;
  try { body = await res.json(); } catch (_) { /* empty body */ }
  if (!res.ok) throw new Error((body && body.error) || res.statusText);
  return body;
}

/* ---------- toasts & button loading ---------- */

function toast(msg, isErr) {
  const host = $("#toastHost");
  const el = document.createElement("div");
  el.className = "toast" + (isErr ? " toast-err" : "");
  el.textContent = msg;
  host.appendChild(el);
  setTimeout(() => {
    el.classList.add("is-leaving");
    setTimeout(() => el.remove(), 350);
  }, 3200);
}

async function withLoading(btn, fn) {
  btn.classList.add("is-loading");
  btn.disabled = true;
  try {
    return await fn();
  } finally {
    btn.classList.remove("is-loading");
    btn.disabled = false;
  }
}

/* ---------- helpers ---------- */

function esc(s) {
  return String(s).replace(/[&<>"']/g, c => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
  }[c]));
}

function stars(n) {
  let out = "";
  for (let i = 1; i <= 5; i++) {
    out += `<span class="${i <= n ? "on" : "off"}">★</span>`;
  }
  return `<span class="stars" aria-label="${n} out of 5 stars">${out}</span>`;
}

function initials(name) {
  return name.split(/\s+/).slice(0, 2).map(w => w[0] || "").join("").toUpperCase();
}

const avatarColors = ["#1e5741", "#7c4a9e", "#b3541e", "#20627d", "#8a2f4e", "#4a6b1f", "#31589e"];
function avatarColor(name) {
  let h = 0;
  for (const ch of name) h = (h * 31 + ch.codePointAt(0)) >>> 0;
  return avatarColors[h % avatarColors.length];
}

function fmtDate(iso) {
  const d = new Date(iso);
  return d.toLocaleDateString("en-IN", { day: "numeric", month: "short", year: "numeric" });
}

function sentiment(rating) {
  if (rating <= 2) return "is-negative";
  if (rating === 3) return "is-neutral";
  return "is-positive";
}

/* ---------- tabs ---------- */

$$(".tab").forEach(tab => {
  tab.addEventListener("click", () => {
    $$(".tab").forEach(t => { t.classList.remove("is-active"); t.setAttribute("aria-selected", "false"); });
    tab.classList.add("is-active");
    tab.setAttribute("aria-selected", "true");
    $$(".panel").forEach(p => p.classList.remove("is-active"));
    $("#panel-" + tab.dataset.tab).classList.add("is-active");
    if (tab.dataset.tab === "digest") loadDigest();
    if (tab.dataset.tab === "outbox") loadOutbox();
    if (tab.dataset.tab === "campaigns") loadCampaigns();
  });
});

/* ---------- connect ---------- */

$("#connectForm").addEventListener("submit", async e => {
  e.preventDefault();
  const btn = $("#connectBtn");
  await withLoading(btn, async () => {
    try {
      const body = JSON.stringify({
        business_name: $("#bizName").value.trim(),
        city: $("#bizCity").value.trim(),
        category: $("#bizCategory").value,
      });
      const res = await api("/profile/connect", { method: "POST", body });
      toast(`Connected — imported ${res.reviews_imported} reviews`);
      state.openDrafts.clear();
      await refreshProfile();
      await loadReviews();
    } catch (err) {
      toast(err.message, true);
    }
  });
});

$("#reconnectBtn").addEventListener("click", () => {
  $("#profileRibbon").hidden = true;
  $("#connectForm").hidden = false;
  $("#bizName").focus();
});

async function refreshProfile() {
  const res = await api("/profile");
  if (!res.connected) {
    state.profile = null;
    renderProfile();
    return;
  }
  state.profile = res.profile;
  state.tracked = res.tracked;
  state.unanswered = res.unanswered;
  renderProfile();
}

function renderProfile() {
  const ribbon = $("#profileRibbon");
  const form = $("#connectForm");
  const toolbar = $("#inboxToolbar");
  if (!state.profile) {
    ribbon.hidden = true;
    form.hidden = false;
    toolbar.hidden = true;
    return;
  }
  const p = state.profile;
  form.hidden = true;
  ribbon.hidden = false;
  toolbar.hidden = false;
  $("#ribbonName").textContent = p.business_name;
  $("#ribbonSub").textContent = `${p.category} · ${p.city} · ${p.phone}`;
  $("#ribbonRating").innerHTML = `${p.rating.toFixed(1)} <span class="star">★</span>`;
  $("#ribbonReviews").textContent = p.review_count;
  $("#ribbonUnanswered").textContent = state.unanswered;
  const badge = $("#inboxCount");
  badge.hidden = state.unanswered === 0;
  badge.textContent = state.unanswered;
}

/* ---------- inbox ---------- */

$$("#filterChips .chip").forEach(chip => {
  chip.addEventListener("click", () => {
    $$("#filterChips .chip").forEach(c => c.classList.remove("is-active"));
    chip.classList.add("is-active");
    state.filter = chip.dataset.filter;
    loadReviews();
  });
});

$("#starFilter").addEventListener("change", e => {
  state.starFilter = e.target.value;
  loadReviews();
});

$$(".seg-btn[data-tone]").forEach(btn => {
  btn.addEventListener("click", () => {
    $$(".seg-btn[data-tone]").forEach(b => b.classList.remove("is-active"));
    btn.classList.add("is-active");
    state.tone = btn.dataset.tone;
  });
});

$$(".seg-btn[data-lang]").forEach(btn => {
  btn.addEventListener("click", () => {
    $$(".seg-btn[data-lang]").forEach(b => b.classList.remove("is-active"));
    btn.classList.add("is-active");
    state.language = btn.dataset.lang;
  });
});

$("#draftAllBtn").addEventListener("click", async e => {
  await withLoading(e.currentTarget, async () => {
    try {
      const res = await api("/reviews/draft-all", {
        method: "POST",
        body: JSON.stringify({ tone: state.tone, language: state.language }),
      });
      toast(`Drafted ${res.drafted} replies in ${state.tone} · ${state.language}`);
      await loadReviews();
      state.reviews.forEach(rv => { if (!rv.replied && rv.draft) state.openDrafts.add(rv.id); });
      renderReviews();
    } catch (err) {
      toast(err.message, true);
    }
  });
});

async function loadReviews() {
  if (!state.profile) { renderReviews(); return; }
  const params = new URLSearchParams();
  if (state.filter) params.set("filter", state.filter);
  if (state.starFilter) params.set("rating", state.starFilter);
  const qs = params.toString();
  const res = await api("/reviews" + (qs ? "?" + qs : ""));
  state.reviews = res.reviews;
  state.counts = res.counts;
  state.unanswered = res.counts.unanswered;
  renderProfile();
  renderReviews();
}

function renderReviews() {
  const host = $("#reviewList");
  if (!state.profile) {
    host.innerHTML = emptyState(
      "inbox",
      "Connect your business to begin",
      "Enter your business name, city and category above — RepPilot pulls in your Google reviews and shows you exactly who is waiting for a reply."
    );
    return;
  }
  if (state.reviews.length === 0) {
    host.innerHTML = emptyState(
      "filter",
      "No reviews match this filter",
      "Try switching back to All, or clear the star filter to see the full inbox."
    );
    return;
  }
  host.innerHTML = state.reviews.map(reviewCard).join("");
  bindReviewCards(host);
}

function reviewCard(rv) {
  const open = state.openDrafts.has(rv.id);
  const badge = rv.replied
    ? '<span class="badge badge-replied">Replied</span>'
    : '<span class="badge badge-unanswered">Needs reply</span>';
  const replyBlock = rv.replied && rv.reply
    ? `<div class="review-reply"><span class="reply-label">Your reply · ${rv.replied_at ? fmtDate(rv.replied_at) : ""}</span>${esc(rv.reply)}</div>`
    : "";
  const actions = rv.replied ? "" : `
    <div class="review-actions">
      <button class="btn btn-ghost btn-sm act-draft" data-id="${rv.id}">${rv.draft ? "Redraft" : "Draft reply"}</button>
      ${rv.draft && !open ? `<button class="btn btn-ghost btn-sm act-open" data-id="${rv.id}">Edit draft</button>` : ""}
    </div>
    <div class="draft-area ${open ? "is-open" : ""}" data-id="${rv.id}">
      <textarea aria-label="Reply draft">${esc(rv.draft || "")}</textarea>
      <div class="draft-foot">
        <button class="btn btn-primary btn-sm act-send" data-id="${rv.id}">Send reply</button>
        <button class="btn btn-ghost btn-sm act-close" data-id="${rv.id}">Close</button>
        <span class="muted">${state.tone} · ${state.language}</span>
      </div>
    </div>`;
  return `
  <article class="review-card ${sentiment(rv.rating)}" data-id="${rv.id}">
    <div class="avatar" style="background:${avatarColor(rv.reviewer)}">${esc(initials(rv.reviewer))}</div>
    <div class="review-body">
      <div class="review-head">
        <span class="review-name">${esc(rv.reviewer)}</span>
        ${stars(rv.rating)}
        <span class="review-date">${fmtDate(rv.date)}</span>
        ${badge}
      </div>
      <p class="review-text">${esc(rv.text)}</p>
      ${replyBlock}
      ${actions}
    </div>
  </article>`;
}

function bindReviewCards(host) {
  $$(".act-draft", host).forEach(btn => btn.addEventListener("click", async () => {
    const id = btn.dataset.id;
    await withLoading(btn, async () => {
      try {
        const res = await api(`/reviews/${id}/draft`, {
          method: "POST",
          body: JSON.stringify({ tone: state.tone, language: state.language }),
        });
        const rv = state.reviews.find(r => r.id === id);
        if (rv) rv.draft = res.draft;
        state.openDrafts.add(id);
        renderReviews();
      } catch (err) {
        toast(err.message, true);
      }
    });
  }));

  $$(".act-open", host).forEach(btn => btn.addEventListener("click", () => {
    state.openDrafts.add(btn.dataset.id);
    renderReviews();
  }));

  $$(".act-close", host).forEach(btn => btn.addEventListener("click", () => {
    state.openDrafts.delete(btn.dataset.id);
    renderReviews();
  }));

  $$(".act-send", host).forEach(btn => btn.addEventListener("click", async () => {
    const id = btn.dataset.id;
    const area = $(`.draft-area[data-id="${id}"] textarea`, host);
    const text = area.value.trim();
    if (!text) { toast("Write or draft a reply first", true); return; }
    await withLoading(btn, async () => {
      try {
        await api(`/reviews/${id}/reply`, { method: "POST", body: JSON.stringify({ reply: text }) });
        state.openDrafts.delete(id);
        toast("Reply sent — review marked as answered");
        await loadReviews();
      } catch (err) {
        toast(err.message, true);
      }
    });
  }));
}

/* ---------- campaigns ---------- */

$("#campaignForm").addEventListener("submit", async e => {
  e.preventDefault();
  const btn = $("#campaignBtn");
  await withLoading(btn, async () => {
    try {
      const res = await api("/campaigns", {
        method: "POST",
        body: JSON.stringify({
          name: $("#campaignName").value.trim(),
          customers: $("#campaignCustomers").value,
        }),
      });
      const skipped = (res.skipped || []).length;
      toast(`Campaign queued — ${res.sent} WhatsApp messages${skipped ? `, ${skipped} lines skipped` : ""}`);
      $("#campaignForm").reset();
      await loadCampaigns();
      await loadOutbox();
    } catch (err) {
      toast(err.message, true);
    }
  });
});

async function loadCampaigns() {
  const res = await api("/campaigns");
  state.campaigns = res.campaigns;
  renderCampaigns();
}

function renderCampaigns() {
  const host = $("#campaignList");
  if (state.campaigns.length === 0) {
    host.innerHTML = emptyState(
      "megaphone",
      "No campaigns yet",
      "Paste a customer list on the left and RepPilot writes each of them a personal WhatsApp review request with your Google review link."
    );
    return;
  }
  host.innerHTML = state.campaigns.map(c => `
    <div class="campaign-item">
      <div>
        <div class="campaign-name">${esc(c.name)}</div>
        <div class="campaign-meta">${fmtDate(c.created_at)} · ${c.customers.length} customers</div>
      </div>
      <span class="campaign-sent">${c.sent} sent to outbox</span>
      ${c.skipped && c.skipped.length ? `<div class="campaign-skip">Skipped ${c.skipped.length} invalid line(s): ${esc(c.skipped.slice(0, 2).join(" · "))}${c.skipped.length > 2 ? " …" : ""}</div>` : ""}
    </div>`).join("");
}

/* ---------- outbox ---------- */

async function loadOutbox() {
  const res = await api("/outbox");
  state.outbox = res.messages;
  renderOutbox();
}

function renderOutbox() {
  const host = $("#outboxList");
  const badge = $("#outboxCount");
  badge.hidden = state.outbox.length === 0;
  badge.textContent = state.outbox.length;
  if (state.outbox.length === 0) {
    host.innerHTML = emptyState(
      "send",
      "Outbox is empty",
      "Messages from review-request campaigns and digest sends land here. In live mode they would go out via WhatsApp (AiSensy)."
    );
    return;
  }
  host.innerHTML = state.outbox.map(m => `
    <div class="wa-msg">
      <div class="wa-head">
        <span class="wa-to">${esc(m.name)}</span>
        <span class="wa-phone">${esc(m.to)}</span>
        <span class="wa-kind">${esc(m.kind)}</span>
      </div>
      <div class="wa-body">${esc(m.body)}</div>
      <div class="wa-foot"><span class="wa-status">${esc(m.status)} · ${fmtDate(m.created_at)}</span></div>
    </div>`).join("");
}

/* ---------- digest ---------- */

async function loadDigest() {
  const host = $("#digestContent");
  if (!state.profile) {
    host.innerHTML = emptyState(
      "report",
      "Your weekly report awaits",
      "Connect your business first — then RepPilot compiles rating trends, response rate and a competitor check into a weekly digest."
    );
    return;
  }
  try {
    state.digest = await api("/digest");
    renderDigest();
  } catch (err) {
    toast(err.message, true);
  }
}

function trendSVG(trend) {
  const W = 320, H = 96, pad = 6, baseY = 78;
  const bw = (W - pad * 2) / trend.length;
  let bars = "";
  trend.forEach((m, i) => {
    const x = pad + i * bw;
    const has = m.count > 0;
    const h = has ? (m.avg_rating / 5) * 56 : 3;
    const cls = !has ? "is-empty" : (m.avg_rating < 3.5 ? "is-low" : "");
    bars += `<rect class="trend-bar ${cls}" x="${(x + 2.5).toFixed(1)}" y="${(baseY - h).toFixed(1)}" width="${(bw - 5).toFixed(1)}" height="${h.toFixed(1)}" rx="1.5"></rect>`;
    if (has) {
      bars += `<text class="trend-value" x="${(x + bw / 2).toFixed(1)}" y="${(baseY - h - 3).toFixed(1)}" text-anchor="middle">${m.avg_rating.toFixed(1)}</text>`;
    }
    const label = m.month.split(" ")[0];
    bars += `<text class="trend-label" x="${(x + bw / 2).toFixed(1)}" y="${baseY + 9}" text-anchor="middle">${label}</text>`;
  });
  return `<svg class="trend-chart" viewBox="0 0 ${W} ${H}" role="img" aria-label="Monthly average rating for the last 12 months">${bars}</svg>`;
}

function renderDigest() {
  const d = state.digest;
  const rows = [`
    <tr class="you">
      <td>${esc(d.business)} <span class="muted">(you)</span></td>
      <td class="num">${d.rating.toFixed(1)} ★</td>
      <td class="num">${d.review_count}</td>
      <td class="num">—</td>
    </tr>`];
  d.competitors.forEach(c => {
    const delta = c.delta > 0
      ? `<span class="delta-up">+${c.delta.toFixed(1)} you lead</span>`
      : c.delta < 0
        ? `<span class="delta-down">${c.delta.toFixed(1)} behind</span>`
        : `<span class="muted">even</span>`;
    rows.push(`
    <tr>
      <td>${esc(c.name)}</td>
      <td class="num">${c.rating.toFixed(1)} ★</td>
      <td class="num">${c.review_count}</td>
      <td class="num">${delta}</td>
    </tr>`);
  });

  $("#digestContent").innerHTML = `
  <div class="digest-sheet">
    <div class="digest-masthead">
      <div class="kicker">The RepPilot Weekly</div>
      <h1>${esc(d.business)}</h1>
      <div class="dateline">${esc(d.category)} · ${esc(d.city)} · week of ${esc(d.week_of)}</div>
    </div>

    <div class="digest-hero">
      <div class="digest-rating">
        <div class="big">${d.rating.toFixed(1)}</div>
        ${stars(Math.round(d.rating))}
      </div>
      <div class="digest-kpis">
        <div class="kpi"><div class="kpi-num">${d.review_count}</div><div class="kpi-label">Lifetime reviews</div></div>
        <div class="kpi"><div class="kpi-num ${d.unanswered > 0 ? "warn" : ""}">${d.unanswered}</div><div class="kpi-label">Unanswered</div></div>
        <div class="kpi"><div class="kpi-num">${d.response_rate.toFixed(1)}%</div><div class="kpi-label">Response rate</div></div>
      </div>
    </div>

    <div class="digest-rule">Rating trend · last 12 months</div>
    ${trendSVG(d.trend)}

    <div class="digest-rule">How you compare nearby</div>
    <div class="digest-table-wrap">
      <table class="digest-table">
        <thead><tr><th>Business</th><th style="text-align:right">Rating</th><th style="text-align:right">Reviews</th><th style="text-align:right">Gap</th></tr></thead>
        <tbody>${rows.join("")}</tbody>
      </table>
    </div>

    <div class="digest-send">
      <div class="field">
        <label for="digestPhone">Send this digest to my WhatsApp</label>
        <input id="digestPhone" placeholder="+91-9812345678" inputmode="tel">
      </div>
      <button class="btn btn-primary" id="digestSendBtn">Send to my WhatsApp</button>
    </div>
    <div class="digest-foot">Compiled by RepPilot autopilot · plan ${esc(d.plan_price)} · data anchored to ${esc(d.week_of)}</div>
  </div>`;

  $("#digestSendBtn").addEventListener("click", async e => {
    const phone = $("#digestPhone").value.trim();
    if (!phone) { toast("Enter your WhatsApp number first", true); return; }
    await withLoading(e.currentTarget, async () => {
      try {
        await api("/digest/send", { method: "POST", body: JSON.stringify({ phone }) });
        toast("Digest queued to your WhatsApp — check the Outbox");
        await loadOutbox();
      } catch (err) {
        toast(err.message, true);
      }
    });
  });
}

/* ---------- empty states ---------- */

const emptyIcons = {
  inbox: '<svg viewBox="0 0 96 96" fill="none"><rect x="14" y="26" width="68" height="48" rx="8" stroke="#1e5741" stroke-width="3"/><path d="M14 50h20l6 9h16l6-9h20" stroke="#1e5741" stroke-width="3"/><path d="M40 12l8 8 8-8" stroke="#eba413" stroke-width="3" stroke-linecap="round"/><path d="M48 20V4" stroke="#eba413" stroke-width="3" stroke-linecap="round"/></svg>',
  filter: '<svg viewBox="0 0 96 96" fill="none"><path d="M16 22h64L56 52v22l-16 8V52L16 22z" stroke="#1e5741" stroke-width="3" stroke-linejoin="round"/><circle cx="72" cy="70" r="13" stroke="#eba413" stroke-width="3"/><path d="M66 70h12" stroke="#eba413" stroke-width="3" stroke-linecap="round"/></svg>',
  megaphone: '<svg viewBox="0 0 96 96" fill="none"><path d="M14 42v14l10 2 6 22 10-3-5-18 37 12V19L24 40l-10 2z" stroke="#1e5741" stroke-width="3" stroke-linejoin="round"/><path d="M80 38c4 2 6 5 6 9s-2 7-6 9" stroke="#eba413" stroke-width="3" stroke-linecap="round"/></svg>',
  send: '<svg viewBox="0 0 96 96" fill="none"><path d="M10 48l72-30-18 60-20-16-12 14-4-22-18-6z" stroke="#1e5741" stroke-width="3" stroke-linejoin="round"/><path d="M44 62l38-44" stroke="#eba413" stroke-width="3" stroke-linecap="round"/></svg>',
  report: '<svg viewBox="0 0 96 96" fill="none"><rect x="22" y="12" width="52" height="72" rx="4" stroke="#1e5741" stroke-width="3"/><path d="M32 30h32M32 42h32M32 54h18" stroke="#1e5741" stroke-width="3" stroke-linecap="round"/><path d="M56 66l6 6 12-14" stroke="#eba413" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"/></svg>',
};

function emptyState(icon, title, text) {
  return `<div class="empty">${emptyIcons[icon] || ""}<h3>${esc(title)}</h3><p>${esc(text)}</p></div>`;
}

/* ---------- boot ---------- */

(async function boot() {
  try {
    await refreshProfile();
    if (state.profile) await loadReviews();
    else renderReviews();
    await loadCampaigns();
    await loadOutbox();
  } catch (err) {
    toast("Could not reach the RepPilot server: " + err.message, true);
  }
})();
