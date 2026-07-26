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
import CalmLoader from "./CalmLoader";
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

function throwIfClerkError(result) {
  if (result.error) throw result.error;
}

function navigateHome({ decorateUrl }) {
  window.location.assign(decorateUrl("/"));
}

export default function AuthPage({
  mode = "sign-in",
  status = "",
  statusDetail = "This will only take a moment.",
  statusKind = "loading",
}) {
  if (status && statusKind === "loading") {
    return <CalmLoader label={status} detail={statusDetail} />;
  }

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
            <AuthStatus message={status} detail={statusDetail} kind={statusKind} />
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

function AuthStatus({ message, detail, kind }) {
  return (
    <div className={`auth-status auth-status-${kind}`} role={kind === "error" ? "alert" : "status"}>
      {kind === "loading" && <LoaderCircle className="auth-spin" size={24} />}
      <strong>{message}</strong>
      <p>{detail}</p>
    </div>
  );
}

function SignInFlow() {
  const { signIn, fetchStatus } = useSignIn();
  const [step, setStep] = useState("credentials");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [verificationStrategy, setVerificationStrategy] = useState("");
  const [error, setError] = useState("");
  const busy = fetchStatus === "fetching";

  const continueWithGoogle = async () => {
    setError("");
    try {
      throwIfClerkError(await signIn.sso({
        strategy: "oauth_google",
        redirectUrl: "/",
        redirectCallbackUrl: "/sso-callback",
      }));
    } catch (requestError) {
      setError(clerkError(requestError));
    }
  };

  const completeSignIn = async () => {
    if (signIn.status === "complete") {
      throwIfClerkError(await signIn.finalize({ navigate: navigateHome }));
      return true;
    }
    return false;
  };

  const prepareSecondFactor = async () => {
    const emailFactor = signIn.supportedSecondFactors.find(
      (factor) => factor.strategy === "email_code",
    );
    const totpFactor = signIn.supportedSecondFactors.find(
      (factor) => factor.strategy === "totp",
    );
    if (emailFactor) {
      throwIfClerkError(await signIn.mfa.sendEmailCode());
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
    setError("");
    try {
      throwIfClerkError(await signIn.password({
        emailAddress: email.trim(),
        password,
      }));
      if (await completeSignIn()) return;
      if (signIn.status === "needs_second_factor" || signIn.status === "needs_client_trust") {
        await prepareSecondFactor();
      } else {
        throw new Error("We couldn’t complete your sign in. Please try another method.");
      }
    } catch (requestError) {
      setError(clerkError(requestError));
    }
  };

  const submitSecondFactor = async (event) => {
    event.preventDefault();
    setError("");
    try {
      const result = verificationStrategy === "totp"
        ? await signIn.mfa.verifyTOTP({ code: code.trim() })
        : await signIn.mfa.verifyEmailCode({ code: code.trim() });
      throwIfClerkError(result);
      if (!(await completeSignIn())) {
        throw new Error("That code could not be verified.");
      }
    } catch (requestError) {
      setError(clerkError(requestError));
    }
  };

  const requestReset = async (event) => {
    event.preventDefault();
    setError("");
    try {
      throwIfClerkError(await signIn.create({ identifier: email.trim() }));
      throwIfClerkError(await signIn.resetPasswordEmailCode.sendCode());
      setCode("");
      setStep("reset-code");
    } catch (requestError) {
      setError(clerkError(requestError));
    }
  };

  const verifyResetCode = async (event) => {
    event.preventDefault();
    setError("");
    try {
      throwIfClerkError(await signIn.resetPasswordEmailCode.verifyCode({
        code: code.trim(),
      }));
      if (signIn.status !== "needs_new_password") {
        throw new Error("That code could not be verified.");
      }
      setStep("reset-password");
    } catch (requestError) {
      setError(clerkError(requestError));
    }
  };

  const saveNewPassword = async (event) => {
    event.preventDefault();
    setError("");
    try {
      throwIfClerkError(await signIn.resetPasswordEmailCode.submitPassword({
        password: newPassword,
      }));
      if (await completeSignIn()) return;
      if (signIn.status === "needs_second_factor" || signIn.status === "needs_client_trust") {
        await prepareSecondFactor();
      } else {
        throw new Error("Your password was changed, but sign in could not be completed.");
      }
    } catch (requestError) {
      setError(clerkError(requestError));
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
      <GoogleAuthButton busy={busy} onClick={continueWithGoogle}>
        Continue with Google
      </GoogleAuthButton>
      <AuthDivider />
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
      <SubmitButton busy={busy}>Sign in</SubmitButton>
    </form>
  );
}

function SignUpFlow() {
  const { signUp, fetchStatus } = useSignUp();
  const [step, setStep] = useState("details");
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [error, setError] = useState("");
  const busy = fetchStatus === "fetching";

  const continueWithGoogle = async () => {
    setError("");
    try {
      throwIfClerkError(await signUp.sso({
        strategy: "oauth_google",
        redirectUrl: "/",
        redirectCallbackUrl: "/sso-callback",
      }));
    } catch (requestError) {
      setError(clerkError(requestError));
    }
  };

  const completeSignUp = async () => {
    if (signUp.status !== "complete") return false;
    throwIfClerkError(await signUp.finalize({ navigate: navigateHome }));
    return true;
  };

  const submitDetails = async (event) => {
    event.preventDefault();
    setError("");
    const names = name.trim().split(/\s+/);
    try {
      throwIfClerkError(await signUp.password({
        firstName: names[0] || undefined,
        lastName: names.slice(1).join(" ") || undefined,
        emailAddress: email.trim(),
        password,
      }));
      if (await completeSignUp()) return;
      throwIfClerkError(await signUp.verifications.sendEmailCode());
      setStep("verify");
    } catch (requestError) {
      setError(clerkError(requestError));
    }
  };

  const verifyEmail = async (event) => {
    event.preventDefault();
    setError("");
    try {
      throwIfClerkError(await signUp.verifications.verifyEmailCode({
        code: code.trim(),
      }));
      if (!(await completeSignUp())) {
        throw new Error("That code could not be verified.");
      }
    } catch (requestError) {
      setError(clerkError(requestError));
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
      <GoogleAuthButton busy={busy} onClick={continueWithGoogle}>
        Continue with Google
      </GoogleAuthButton>
      <AuthDivider />
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
      <SubmitButton busy={busy}>Create account</SubmitButton>
      <p className="auth-terms">
        Your learning space is private by default. By continuing, you agree to
        our <a href="/terms">Terms</a> and acknowledge our{" "}
        <a href="/privacy">Privacy Policy</a>.
      </p>
    </form>
  );
}

function GoogleAuthButton({ busy, onClick, children }) {
  return (
    <button
      className="auth-google"
      type="button"
      onClick={onClick}
      disabled={busy}
    >
      <GoogleMark />
      <span>{busy ? "Opening Google…" : children}</span>
      <ArrowRight size={15} />
    </button>
  );
}

function GoogleMark() {
  return (
    <svg
      aria-hidden="true"
      className="auth-google-mark"
      viewBox="0 0 24 24"
    >
      <path fill="#4285F4" d="M21.6 12.23c0-.71-.06-1.4-.18-2.07H12v3.92h5.38a4.6 4.6 0 0 1-2 3.02v2.54h3.24c1.9-1.74 2.98-4.31 2.98-7.41Z" />
      <path fill="#34A853" d="M12 22c2.7 0 4.98-.9 6.63-2.43l-3.24-2.54c-.9.6-2.05.96-3.39.96-2.61 0-4.82-1.76-5.61-4.13H3.04v2.62A10 10 0 0 0 12 22Z" />
      <path fill="#FBBC05" d="M6.39 13.86A6.02 6.02 0 0 1 6.08 12c0-.65.11-1.28.31-1.86V7.52H3.04A10 10 0 0 0 2 12c0 1.61.39 3.14 1.04 4.48l3.35-2.62Z" />
      <path fill="#EA4335" d="M12 6.01c1.47 0 2.78.5 3.82 1.5l2.88-2.88A9.65 9.65 0 0 0 12 2a10 10 0 0 0-8.96 5.52l3.35 2.62C7.18 7.77 9.39 6.01 12 6.01Z" />
    </svg>
  );
}

function AuthDivider() {
  return (
    <div className="auth-divider" role="separator">
      <span>or continue with email</span>
    </div>
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
