const navItems = [
  ["home", "/web/index.html", "Home"],
  ["how", "/web/how-it-works.html", "How It Works"],
  ["protocol", "/web/protocol.html", "Protocol"],
  ["evidence", "/web/evidence.html", "Demo Evidence"],
  ["security", "/web/security.html", "Security"],
  ["developers", "/web/developers.html", "Developers"],
  ["audit", "/web/audit.html", "Audit Readiness"],
  ["faq", "/web/faq.html", "FAQ"]
];

const explorerBase = "https://coston2-explorer.flare.network";

class SiteHeader extends HTMLElement {
  connectedCallback() {
    const active = document.body.dataset.page || "home";
    const links = navItems
      .map(([key, href, label]) => {
        const current = key === active ? ' aria-current="page"' : "";
        return `<a href="${href}"${current}>${label}</a>`;
      })
      .join("");

    this.innerHTML = `
      <header class="site-header">
        <a class="brand-link" href="/web/index.html" aria-label="WorkProof home">
          <img src="/assets/workproof-logo.png" alt="" />
          <span>
            <strong>WorkProof</strong>
            <small>FCC-verified escrow</small>
          </span>
        </a>
        <button class="nav-toggle" type="button" aria-expanded="false" aria-controls="primary-nav">
          <span></span><span></span><span></span>
        </button>
        <nav id="primary-nav" class="primary-nav" aria-label="Primary navigation">
          ${links}
        </nav>
      </header>
    `;

    const toggle = this.querySelector(".nav-toggle");
    const nav = this.querySelector(".primary-nav");
    toggle.addEventListener("click", () => {
      const isOpen = toggle.getAttribute("aria-expanded") === "true";
      toggle.setAttribute("aria-expanded", String(!isOpen));
      nav.classList.toggle("open", !isOpen);
    });
  }
}

class SiteFooter extends HTMLElement {
  connectedCallback() {
    this.innerHTML = `
      <footer class="site-footer">
        <div>
          <strong>WorkProof</strong>
          <p>Coston2 FCC escrow demo. Simulated attestation. Audit required before production.</p>
        </div>
        <nav aria-label="Footer links">
          <a href="https://github.com/cyberrockng/workproof">Repository</a>
          <a href="/README.md">Docs</a>
          <a href="/docs/evidence/demo-run.json">Evidence</a>
          <a href="/docs/evidence/AUDIT_RESPONSE.md">Security</a>
        </nav>
      </footer>
    `;
  }
}

class PageHero extends HTMLElement {
  connectedCallback() {
    const eyebrow = this.getAttribute("eyebrow") || "";
    const title = this.getAttribute("title") || "";
    const body = this.innerHTML;
    this.innerHTML = `
      <section class="page-hero">
        <div>
          <p class="eyebrow">${eyebrow}</p>
          <h1>${title}</h1>
          <p class="lead">${body}</p>
        </div>
        <div class="hero-badges">
          <span>Coston2 Demo</span>
          <span>Simulated Attestation</span>
          <span>Not Mainnet Ready</span>
          <span>Audit Required Before Production</span>
        </div>
      </section>
    `;
  }
}

class ProtocolStep extends HTMLElement {
  connectedCallback() {
    const number = this.getAttribute("number") || "";
    const title = this.getAttribute("title") || "";
    const body = this.innerHTML;
    this.innerHTML = `
      <article class="step-card">
        <span class="step-number">${number}</span>
        <h3>${title}</h3>
        <p>${body}</p>
      </article>
    `;
  }
}

class EvidenceCard extends HTMLElement {
  connectedCallback() {
    const title = this.getAttribute("title") || "Evidence";
    const label = this.getAttribute("label") || "";
    const tx = this.getAttribute("tx") || "";
    const tone = this.getAttribute("tone") || "info";
    const href = `${explorerBase}/tx/${tx}`;
    this.innerHTML = `
      <article class="evidence-card ${tone}">
        <span class="evidence-label">${label}</span>
        <h3>${title}</h3>
        <code>${tx}</code>
        <a href="${href}" target="_blank" rel="noreferrer">Open transaction</a>
      </article>
    `;
  }
}

customElements.define("site-header", SiteHeader);
customElements.define("site-footer", SiteFooter);
customElements.define("page-hero", PageHero);
customElements.define("protocol-step", ProtocolStep);
customElements.define("evidence-card", EvidenceCard);
