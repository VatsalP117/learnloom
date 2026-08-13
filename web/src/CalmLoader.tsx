import BrandMark from "./BrandMark";

type CalmLoaderProps = {
  label?: string;
  detail?: string;
  variant?: "screen" | "inline";
};

export default function CalmLoader({
  label = "Settling into your learning space…",
  detail = "Finding your place, quietly.",
  variant = "screen",
}: CalmLoaderProps) {
  const Element = variant === "screen" ? "main" : "div";

  return (
    <Element
      className={`calm-loader calm-loader-${variant}`}
      role="status"
      aria-live="polite"
      aria-label={label}
    >
      <div className="calm-loader-content">
        <span className="calm-loader-mark" aria-hidden="true">
          <BrandMark />
        </span>
        <strong>{label}</strong>
        {detail ? <span className="calm-loader-detail">{detail}</span> : null}
      </div>
    </Element>
  );
}
