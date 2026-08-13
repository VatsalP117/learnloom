const brandMarkStyle = {
  width: "100%",
  height: "100%",
  display: "block",
  objectFit: "cover" as const,
  borderRadius: "25%",
};

export default function BrandMark() {
  return (
    <img
      src="/favicon.svg"
      alt=""
      aria-hidden="true"
      style={brandMarkStyle}
    />
  );
}
