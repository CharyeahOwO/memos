import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import ReactMarkdown from "react-markdown";
import rehypeKatex from "rehype-katex";
import rehypeRaw from "rehype-raw";
import rehypeSanitize from "rehype-sanitize";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import { describe, expect, it } from "vitest";
import { allowAnyUrlTransform, isTrustedIframeSrc, SANITIZE_SCHEMA } from "@/components/MemoContent/constants";
import { TrustedIframe } from "@/components/MemoContent/TrustedIframe";

const renderMemoContent = (content: string): string =>
  renderToStaticMarkup(
    <ReactMarkdown
      remarkPlugins={[remarkMath]}
      rehypePlugins={[rehypeRaw, [rehypeSanitize, SANITIZE_SCHEMA], [rehypeKatex, { throwOnError: false, strict: false }]]}
      components={{ iframe: TrustedIframe }}
      urlTransform={allowAnyUrlTransform}
    >
      {content}
    </ReactMarkdown>,
  );

const renderGfmContent = (content: string): string =>
  renderToStaticMarkup(
    <ReactMarkdown remarkPlugins={[remarkGfm]} rehypePlugins={[[rehypeSanitize, SANITIZE_SCHEMA]]}>
      {content}
    </ReactMarkdown>,
  );

describe("memo content sanitization", () => {
  it("strips user-controlled inline styles from raw HTML spans", () => {
    const html = renderMemoContent('<span style="position:fixed;inset:0;z-index:99999">overlay</span>');

    expect(html).toMatch(/<span>overlay<\/span>/);
    expect(html).not.toMatch(/style=/);
    expect(html).not.toMatch(/position:fixed/);
  });

  it("still renders KaTeX output after sanitizing math marker classes", () => {
    const html = renderMemoContent("$L$");

    expect(html).toMatch(/class="katex"/);
    expect(html).toMatch(/class="katex-html"/);
  });

  it("preserves checked state for GFM task list items", () => {
    const html = renderGfmContent("- [x] Done\n- [ ] Todo");
    const inputs = html.match(/<input[^>]+\/>/g) ?? [];

    expect(inputs).toHaveLength(2);
    expect(inputs[0]).toContain('checked=""');
    expect(inputs[1]).not.toContain('checked=""');
  });
});

describe("trusted iframe providers", () => {
  it("accepts any non-empty iframe src", () => {
    expect(isTrustedIframeSrc("https://www.youtube.com/embed/abc123")).toBe(true);
    expect(isTrustedIframeSrc("https://i.y.qq.com/n2/m/outchain/player/index.html?songid=374312821&songtype=0")).toBe(true);
    expect(isTrustedIframeSrc("http://example.test/embed/abc123")).toBe(true);
    expect(isTrustedIframeSrc("javascript:alert(1)")).toBe(true);
    expect(isTrustedIframeSrc("")).toBe(false);
  });

  it("renders iframe embeds from arbitrary providers and protocols", () => {
    const qqMusic = renderMemoContent(
      '<iframe src="https://i.y.qq.com/n2/m/outchain/player/index.html?songid=374312821&amp;songtype=0" title="demo"></iframe>',
    );
    const javascriptUrl = renderMemoContent('<iframe src="javascript:alert(1)" title="demo"></iframe>');

    expect(qqMusic).toMatch(/<iframe/);
    expect(qqMusic).toMatch(/i\.y\.qq\.com\/n2\/m\/outchain\/player\/index\.html/);
    expect(javascriptUrl).toMatch(/<iframe/);
    expect(javascriptUrl).toMatch(/javascript:alert\(1\)/);
  });
});
