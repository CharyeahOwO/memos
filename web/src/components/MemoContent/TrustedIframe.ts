import { createElement } from "react";
import { cn } from "@/lib/utils";
import { isTrustedIframeSrc } from "./constants";

const escapeAttribute = (value: unknown) =>
  String(value).replace(/&/g, "&amp;").replace(/"/g, "&quot;").replace(/</g, "&lt;").replace(/>/g, "&gt;");

const attr = (name: string, value: unknown) => {
  if (value === undefined || value === null || value === false) {
    return undefined;
  }
  if (value === true) {
    return name;
  }
  return `${name}="${escapeAttribute(value)}"`;
};

export const TrustedIframe = (props: React.ComponentProps<"iframe">) => {
  if (typeof props.src !== "string" || !isTrustedIframeSrc(props.src)) {
    return null;
  }

  const rawProps = props as Record<string, unknown>;
  const className = cn("max-w-full rounded-lg border border-border", props.className);
  const attributes = [
    attr("src", props.src),
    attr("width", props.width),
    attr("height", props.height),
    attr("frameborder", rawProps.frameborder ?? rawProps.frameBorder),
    attr("allowfullscreen", rawProps.allowfullscreen ?? props.allowFullScreen),
    attr("allow", props.allow),
    attr("title", props.title),
    attr("referrerpolicy", rawProps.referrerpolicy ?? props.referrerPolicy),
    attr("loading", props.loading),
    attr("class", className),
  ].filter(Boolean);

  return createElement("span", {
    dangerouslySetInnerHTML: {
      __html: `<iframe ${attributes.join(" ")}></iframe>`,
    },
  });
};
