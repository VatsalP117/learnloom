import { initializePaddle, type Paddle } from "@paddle/paddle-js";
import { useCallback, useEffect, useState } from "react";

interface PaddlePublicConfig {
  commerceAvailable: boolean;
  environment?: "sandbox" | "production";
  clientToken?: string;
}

let paddleInitialization: Promise<Paddle | undefined> | null = null;

export default function PaddleCheckoutPage() {
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState("");
  const transactionId = new URLSearchParams(window.location.search).get("_ptxn");

  const startCheckout = useCallback(async () => {
    setState("loading");
    setError("");
    if (!transactionId?.startsWith("txn_")) {
      setError("This checkout link is incomplete. Return to plan settings and start again.");
      setState("error");
      return;
    }
    try {
      const response = await fetch("/api/billing/config", {
        credentials: "same-origin",
        headers: { Accept: "application/json" },
      });
      if (!response.ok) throw new Error("Checkout configuration is unavailable.");
      const config = await response.json() as PaddlePublicConfig;
      if (!config.commerceAvailable || !config.clientToken || !config.environment) {
        throw new Error("Checkout is not available in this environment yet.");
      }
      if (!paddleInitialization) {
        paddleInitialization = initializePaddle({
          token: config.clientToken,
          environment: config.environment,
          checkout: {
            settings: {
              displayMode: "overlay",
              theme: "light",
              locale: "en",
              variant: "one-page",
            },
          },
          eventCallback: (event) => {
            if (event.name === "checkout.completed") {
              window.location.assign("/settings?checkout=complete");
            }
          },
        });
      }
      const paddle = await paddleInitialization;
      if (!paddle) throw new Error("Paddle Checkout could not be initialized.");
      setState("ready");
    } catch (checkoutError) {
      paddleInitialization = null;
      setError(checkoutError instanceof Error
        ? checkoutError.message
        : "Checkout could not be opened.");
      setState("error");
    }
  }, [transactionId]);

  useEffect(() => {
    void startCheckout();
  }, [startCheckout]);

  return (
    <main className="auth-shell">
      <section className="claim-card" aria-live="polite">
        <p>Learnloom secure checkout</p>
        <h1>{state === "error" ? "Checkout needs attention" : "Opening Paddle Checkout…"}</h1>
        <p>
          {state === "ready"
            ? "Your secure payment window is open. You can return here if you close it."
            : error || "Taxes and the renewal total are shown before you confirm."}
        </p>
        {state === "error" ? <button type="button" onClick={() => void startCheckout()}>Try again</button> : null}
        <a href="/settings">Back to plan settings</a>
      </section>
    </main>
  );
}
