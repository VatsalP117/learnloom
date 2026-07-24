export const rootDomain =
  import.meta.env.VITE_LEARNLOOM_ROOT_DOMAIN?.trim().toLowerCase() ||
  "learnloom.blog";

export const appOrigin = `https://app.${rootDomain}`;

export function personalSiteHost(username: string) {
  return `${username}.${rootDomain}`;
}
