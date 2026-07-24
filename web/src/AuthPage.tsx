import {
  ArrowLeft,
  ArrowRight,
  Eye,
  EyeOff,
  Leaf,
  LoaderCircle,
  Sparkles,
} from "lucide-react";
import { useState } from "react";
import { useSignIn, useSignUp } from "@clerk/react";
import "./auth.css";

const AUTH_COPY = {
  "sign-in": {
    eyebrow: "Welcome back",
    title: "Return to your learning.",
    description: "Your streams, lessons, and ideas are right where you left them.",
    alternate: "New to Learnloom?",
    alternateAction: "Create an account",
    alternateHref: "/sign-up",
  },
  "sign-up": {
    eyebrow: "Begin your practice",
    title: "Make space for what matters.",
    description: "Create a calm, personal rhythm for learning things deeply.",
    alternate: "Already have an account?",
    alternateAction: "Sign in",
    alternateHref: "/sign-in",
  },
};

function clerkError(error) {
  return (
    error?.errors?.[0]?.longMessage ||
    error?.errors?.[0]?.message ||
    error?.message ||
    "Something went wrong. Please try again."
  );
}

export default function AuthPage({ mode = "sign-in", status = "" }) {
  const copy = AUTH_COPY[mode] ?? AUTH_COPY["sign-in"];

  return (
    <main className="custom-auth-shell">
      <section className="auth-visual" aria-label="A quiet mountain landscape">
        <div className="auth-visual-shade" />
        <a className="auth-visual-brand" href="/marketing" aria-label="Learnloom home">
          <span><Sparkles size={16} strokeWidth={1.8} /></span>
          <strong>Learnloom</strong>
        </a>
        <div className="auth-visual-copy">
          <span className="auth-visual-kicker"><Leaf size={13} /> A quieter place to learn</span>
          <blockquote>
            Go deep on what matters. Let the rest stay quiet.
          </blockquote>
          <p>Build understanding, one thoughtful lesson at a time.</p>
        </div>
        <span className="auth-visual-index">01 / Learn at your own rhythm</span>
      </section>

      <section className="auth-panel">
        <a className="auth-mobile-brand" href="/marketing">
          <span><Sparkles size={15} /></span>
          Learnloom
        </a>
        <div className="auth-panel-inner">
          {status ? (
            <AuthStatus message={status} />
          ) : mode === "sign-up" ? (
            <SignUpFlow />
          ) : (
            <SignInFlow />
          )}

          {!status && (
            <div className="auth-heading">
              <span>{copy.eyebrow}</span>
              <h1>{copy.title}</h1>
              <p>{copy.description}</p>
            </div>
          )}
        </div>

        {!status && (
          <p className="auth-switch">
            {copy.alternate} <a href={copy.alternateHref}>{copy.alternateAction}</a>
          </p>
        )}
      </section>
    </main>
  );
}

function AuthStatus({ message }) {
  return (
    <div className="auth-status" role="status">
      <LoaderCircle className="auth-spin" size={24} />
      <strong>{message}</strong>
      <p>This will only take a moment.</p>
    </div>
  );
}

function SignInFlow() {
  // Clerk's compatibility flow is broader than the signal-based hook declaration.
  const { isLoaded, signIn, setActive } = useSignIn() as any;
  const [step, setStep] = useState("credentials");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [verificationStrategy, setVerificationStrategy] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const completeSignIn = async (result) => {
    if (result.status === "complete" && result.createdSessionId) {
      await setActive({ session: result.createdSessionId, redirectUrl: "/" });
      return true;
    }
    return false;
  };

  const prepareSecondFactor = async (result) => {
    const emailFactor = result.supportedSecondFactors?.find(
      (factor) => factor.strategy === "email_code",
    );
    const totpFactor = result.supportedSecondFactors?.find(
      (factor) => factor.strategy === "totp",
    );
    if (emailFactor) {
      await signIn.prepareSecondFactor({
        strategy: "email_code",
        emailAddressId: emailFactor.emailAddressId,
      });
      setVerificationStrategy("email_code");
      setStep("second-factor");
      return;
    }
    if (totpFactor) {
      setVerificationStrategy("totp");
      setStep("second-factor");
      return;
    }
    throw new Error("This account needs an additional sign-in method that is not available here.");
  };

  const submitCredentials = async (event) => {
    event.preventDefault();
    if (!isLoaded) return;
    setBusy(true);
    setError("");
    try {
      const result = await signIn.create({
        identifier: email.trim(),
        password,
        strategy: "password",
      });
      if (await completeSignIn(result)) return;
      if (result.status === "needs_second_factor") {
        await prepareSecondFactor(result);
      } else {
        throw new Error("We couldn’t complete your sign in. Please try another method.");
      }
    } catch (requestError) {
      setError(clerkError(requestError));
    } finally {
      setBusy(false);
    }
  };

  const submitSecondFactor = async (event) => {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const result = await signIn.attemptSecondFactor({
        strategy: verificationStrategy,
        code: code.trim(),
      });
      if (!(await completeSignIn(result))) {
        throw new Error("That code could not be verified.");
      }
    } catch (requestError) {
      setError(clerkError(requestError));
    } finally {
      setBusy(false);
    }
  };

  const requestReset = async (event) => {
    event.preventDefault();
    if (!isLoaded) return;
    setBusy(true);
    setError("");
    try {
      await signIn.create({
        strategy: "reset_password_email_code",
        identifier: email.trim(),
      });
      setCode("");
      setStep("reset-code");
    } catch (requestError) {
      setError(clerkError(requestError));
    } finally {
      setBusy(false);
    }
  };

  const verifyResetCode = async (event) => {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const result = await signIn.attemptFirstFactor({
        strategy: "reset_password_email_code",
        code: code.trim(),
      });
      if (result.status !== "needs_new_password") {
        throw new Error("That code could not be verified.");
      }
      setStep("reset-password");
    } catch (requestError) {
      setError(clerkError(requestError));
    } finally {
      setBusy(false);
    }
  };

  const saveNewPassword = async (event) => {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const result = await signIn.resetPassword({ password: newPassword });
      if (await completeSignIn(result)) return;
      if (result.status === "needs_second_factor") {
        await prepareSecondFactor(result);
      } else {
        throw new Error("Your password was changed, but sign in could not be completed.");
      }
    } catch (requestError) {
      setError(clerkError(requestError));
    } finally {
      setBusy(false);
    }
  };

  if (step === "forgot") {
    return (
      <AuthForm
        title="Reset your password."
        description="Enter the email you use for Learnloom and we’ll send you a reset code."
        onSubmit={requestReset}
        error={error}
        busy={busy}
        submitLabel="Send reset code"
        onBack={() => { setStep("credentials"); setError(""); }}
      >
        <TextField
          label="Email address"
          type="email"
          autoComplete="email"
          value={email}
          onChange={setEmail}
          placeholder="you@example.com"
        />
      </AuthForm>
    );
  }

  if (step === "reset-code" || step === "second-factor") {
    const isReset = step === "reset-code";
    return (
      <AuthForm
        title={isReset ? "Check your inbox." : "One more step."}
        description={
          isReset
            ? `We sent a six-digit reset code to ${email}.`
            : verificationStrategy === "totp"
              ? "Enter the code from your authenticator app."
              : "Enter the verification code we sent to your email."
        }
        onSubmit={isReset ? verifyResetCode : submitSecondFactor}
        error={error}
        busy={busy}
        submitLabel="Verify code"
        onBack={() => { setStep("credentials"); setCode(""); setError(""); }}
      >
        <TextField
          label="Verification code"
          inputMode="numeric"
          autoComplete="one-time-code"
          value={code}
          onChange={setCode}
          placeholder="000000"
          code
        />
      </AuthForm>
    );
  }

  if (step === "reset-password") {
    return (
      <AuthForm
        title="Choose a new password."
        description="Make it memorable, secure, and at least eight characters long."
        onSubmit={saveNewPassword}
        error={error}
        busy={busy}
        submitLabel="Save new password"
        onBack={() => { setStep("credentials"); setError(""); }}
      >
        <PasswordField
          label="New password"
          value={newPassword}
          onChange={setNewPassword}
          autoComplete="new-password"
        />
      </AuthForm>
    );
  }

  return (
    <form className="auth-form" onSubmit={submitCredentials}>
      <TextField
        label="Email address"
        type="email"
        autoComplete="email"
        value={email}
        onChange={setEmail}
        placeholder="you@example.com"
      />
      <PasswordField
        label="Password"
        value={password}
        onChange={setPassword}
        autoComplete="current-password"
        action={
          <button type="button" onClick={() => { setStep("forgot"); setError(""); }}>
            Forgot password?
          </button>
        }
      />
      <FormError message={error} />
      <SubmitButton busy={busy || !isLoaded}>Sign in</SubmitButton>
    </form>
  );
}

function SignUpFlow() {
  const { isLoaded, signUp, setActive } = useSignUp() as any;
  const [step, setStep] = useState("details");
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const submitDetails = async (event) => {
    event.preventDefault();
    if (!isLoaded) return;
    setBusy(true);
    setError("");
    const names = name.trim().split(/\s+/);
    try {
      const result = await signUp.create({
        firstName: names[0] || undefined,
        lastName: names.slice(1).join(" ") || undefined,
        emailAddress: email.trim(),
        password,
      });
      if (result.status === "complete" && result.createdSessionId) {
        await setActive({ session: result.createdSessionId, redirectUrl: "/" });
        return;
      }
      await signUp.prepareEmailAddressVerification({ strategy: "email_code" });
      setStep("verify");
    } catch (requestError) {
      setError(clerkError(requestError));
    } finally {
      setBusy(false);
    }
  };

  const verifyEmail = async (event) => {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const result = await signUp.attemptEmailAddressVerification({ code: code.trim() });
      if (result.status !== "complete" || !result.createdSessionId) {
        throw new Error("That code could not be verified.");
      }
      await setActive({ session: result.createdSessionId, redirectUrl: "/" });
    } catch (requestError) {
      setError(clerkError(requestError));
    } finally {
      setBusy(false);
    }
  };

  if (step === "verify") {
    return (
      <AuthForm
        title="Check your inbox."
        description={`We sent a six-digit verification code to ${email}.`}
        onSubmit={verifyEmail}
        error={error}
        busy={busy}
        submitLabel="Create my space"
        onBack={() => { setStep("details"); setCode(""); setError(""); }}
      >
        <TextField
          label="Verification code"
          inputMode="numeric"
          autoComplete="one-time-code"
          value={code}
          onChange={setCode}
          placeholder="000000"
          code
        />
      </AuthForm>
    );
  }

  return (
    <form className="auth-form" onSubmit={submitDetails}>
      <TextField
        label="Your name"
        autoComplete="name"
        value={name}
        onChange={setName}
        placeholder="How should we greet you?"
      />
      <TextField
        label="Email address"
        type="email"
        autoComplete="email"
        value={email}
        onChange={setEmail}
        placeholder="you@example.com"
      />
      <PasswordField
        label="Password"
        value={password}
        onChange={setPassword}
        autoComplete="new-password"
        hint="Use 8 or more characters"
      />
      <div id="clerk-captcha" className="auth-captcha" />
      <FormError message={error} />
      <SubmitButton busy={busy || !isLoaded}>Create account</SubmitButton>
      <p className="auth-terms">
        Your learning space is private by default.
      </p>
    </form>
  );
}

function AuthForm({
  title,
  description,
  onSubmit,
  error,
  busy,
  submitLabel,
  onBack,
  children,
}) {
  return (
    <div className="auth-subflow">
      <button className="auth-back" type="button" onClick={onBack}>
        <ArrowLeft size={15} /> Back
      </button>
      <div className="auth-subflow-heading">
        <h1>{title}</h1>
        <p>{description}</p>
      </div>
      <form className="auth-form" onSubmit={onSubmit}>
        {children}
        <FormError message={error} />
        <SubmitButton busy={busy}>{submitLabel}</SubmitButton>
      </form>
    </div>
  );
}

function TextField({ label, value, onChange, code = false, ...inputProps }) {
  return (
    <label className={`auth-field${code ? " auth-code-field" : ""}`}>
      <span>{label}</span>
      <input
        required
        value={value}
        onChange={(event) => onChange(event.target.value)}
        {...inputProps}
      />
    </label>
  );
}

function PasswordField({
  label,
  value,
  onChange,
  action = null,
  hint = "",
  autoComplete,
}) {
  const [visible, setVisible] = useState(false);
  return (
    <div className="auth-field">
      <span className="auth-field-label">
        <label>{label}</label>
        {action}
      </span>
      <span className="auth-password-wrap">
        <input
          aria-label={label}
          required
          minLength={8}
          type={visible ? "text" : "password"}
          autoComplete={autoComplete}
          value={value}
          onChange={(event) => onChange(event.target.value)}
        />
        <button
          type="button"
          onClick={() => setVisible((current) => !current)}
          aria-label={visible ? "Hide password" : "Show password"}
        >
          {visible ? <EyeOff size={17} /> : <Eye size={17} />}
        </button>
      </span>
      {hint && <small>{hint}</small>}
    </div>
  );
}

function FormError({ message }) {
  if (!message) return null;
  return <p className="auth-error" role="alert">{message}</p>;
}

function SubmitButton({ busy, children }) {
  return (
    <button className="auth-submit" type="submit" disabled={busy}>
      <span>{busy ? "Just a moment…" : children}</span>
      {busy ? <LoaderCircle className="auth-spin" size={17} /> : <ArrowRight size={17} />}
    </button>
  );
}
