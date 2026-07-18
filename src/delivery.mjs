import { renderDossierEmail } from "./render.mjs";

export function createDeliveryAdapters(config, options = {}) {
  return config.deliveries
    .filter((delivery) => delivery.enabled)
    .map((delivery) => {
      if (delivery.kind === "resend") {
        return new ResendDelivery(delivery, options);
      }
      throw new Error(`Unsupported delivery kind: ${delivery.kind}`);
    });
}

export class ResendDelivery {
  constructor(config, options = {}) {
    this.id = config.id;
    this.kind = "resend";
    this.config = config;
    this.environment = options.env ?? process.env;
    this.fetchImpl = options.fetchImpl ?? globalThis.fetch;
    this.endpoint = options.resendEndpoint ?? "https://api.resend.com/emails";
  }

  async deliver({ runId, dossier, markdown }) {
    const apiKey = this.environment[this.config.apiKeyEnv];
    if (!apiKey) {
      throw new Error(
        `Missing Resend credential in environment variable ${this.config.apiKeyEnv}.`,
      );
    }
    const rendered = renderDossierEmail(dossier, markdown);
    const response = await this.fetchImpl(this.endpoint, {
      method: "POST",
      headers: {
        authorization: `Bearer ${apiKey}`,
        "content-type": "application/json",
        "idempotency-key": `learnloom/${runId}/${this.id}`,
      },
      body: JSON.stringify({
        from: this.config.from,
        to: this.config.to,
        subject: `${this.config.subjectPrefix}: ${dossier.title} — ${dossier.date}`,
        html: rendered.html,
        text: rendered.text,
      }),
      signal: AbortSignal.timeout(30_000),
    });
    let payload;
    try {
      payload = await response.json();
    } catch {
      payload = null;
    }
    if (!response.ok) {
      const detail =
        typeof payload?.message === "string"
          ? payload.message
          : typeof payload?.error?.message === "string"
            ? payload.error.message
            : "";
      throw new Error(
        `Resend returned HTTP ${response.status}${detail ? `: ${detail.slice(0, 300)}` : ""}`,
      );
    }
    if (typeof payload?.id !== "string" || payload.id === "") {
      throw new Error("Resend returned no email identifier.");
    }
    return { externalId: payload.id };
  }
}

export function checkDeliveries(config, options = {}) {
  const environment = options.env ?? process.env;
  return config.deliveries
    .filter((delivery) => delivery.enabled)
    .map((delivery) => {
      const configured = Boolean(environment[delivery.apiKeyEnv]);
      return {
        name: `Delivery ${delivery.id}`,
        ok: configured,
        detail: configured
          ? `${delivery.kind} configured`
          : `${delivery.apiKeyEnv} is not set`,
      };
    });
}

