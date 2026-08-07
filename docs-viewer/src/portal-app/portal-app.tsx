import Fuse from "fuse.js";
import { useCallback, useEffect, useId, useMemo, useState } from "react";
import { SearchFieldClient } from "@/components/design-system/form/search-field-client/search-field-client";
import {
  ToggleGroupNative,
  ToggleGroupNativeItem,
} from "@/components/design-system/form/toggle-group-native/toggle-group-native";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/design-system/overlay/dialog/dialog";

import type { DocsJson, PortalItem } from "../docs-json/docs-json";
import { DocumentContent } from "../document-content/document-content";
import { parseHashRoute, resolveActiveGroupSlug } from "../hash-route/hash-route";
import { applyLangFilter, type PortalLang } from "../lang-filter/lang-filter";
import { parseMarkdownDocument } from "../markdown/markdown-document";
import { PortalCardGrid } from "../portal-card-grid/portal-card-grid";
import { PortalSidebar } from "../portal-sidebar/portal-sidebar";
import type { SanitizedDocument } from "../sanitize/sanitized-document";
import { buildSearchCorpus, type SearchEntry, searchKeys } from "../search/search-corpus";

/** 開いている文書。取得中は本文が未確定なので、題だけ先に確定させる。 */
type OpenDocument = {
  name: string;
  content: SanitizedDocument | null;
};

export type PortalAppProps = {
  docs: DocsJson;
};

/**
 * ポータルのビューアー本体。
 *
 * @remarks
 * 表示状態は位置ハッシュが持ちます。静的配信されるため経路をサーバへ問い合わせられず、共有・
 * 履歴・戻る操作に対して復元可能なのはハッシュだけだからです。検索語と表示言語はハッシュに
 * 載せません。どちらも一時的な絞り込みであり、共有したいのは「どの文書を見ているか」だからです。
 */
export function PortalApp({ docs }: PortalAppProps) {
  const [lang, setLang] = useState<PortalLang>("EN");
  const [query, setQuery] = useState("");
  const [hash, setHash] = useState(() => window.location.hash);
  const [openDocument, setOpenDocument] = useState<OpenDocument | null>(null);
  const searchResultsHeadingId = useId();

  const selectEnglish = useCallback(() => setLang("EN"), []);
  const selectJapanese = useCallback(() => setLang("JA"), []);
  const closeDocument = useCallback(() => setOpenDocument(null), []);

  useEffect(() => {
    const onHashChange = () => setHash(window.location.hash);

    window.addEventListener("hashchange", onHashChange);

    return () => window.removeEventListener("hashchange", onHashChange);
  }, []);

  // ハッシュが section を指していれば、その見出しまで送る。group を切り替えるだけでは
  // 長い group の末尾にある section へ辿り着けず、link が指した先と表示がずれる。
  useEffect(() => {
    const { sectionSlug } = parseHashRoute(hash);

    if (!sectionSlug) {
      return;
    }

    document.getElementById(`section-${sectionSlug}`)?.scrollIntoView({ block: "start" });
  }, [hash]);

  const visibleGroups = useMemo(() => applyLangFilter(docs.groups, lang), [docs.groups, lang]);
  const corpus = useMemo(() => buildSearchCorpus(visibleGroups), [visibleGroups]);
  const fuse = useMemo(() => new Fuse(corpus, { keys: [...searchKeys], threshold: 0.3 }), [corpus]);

  const activeGroupSlug = resolveActiveGroupSlug(visibleGroups, parseHashRoute(hash).groupSlug);
  const activeGroup = visibleGroups.find((group) => group.slug === activeGroupSlug) ?? null;

  const results: SearchEntry[] | null = query.trim()
    ? fuse.search(query).map((hit) => hit.item)
    : null;

  const onOpenDocument = useCallback((item: PortalItem) => {
    setOpenDocument({ name: item.name, content: null });

    fetch(item.path)
      .then((response) => {
        if (!response.ok) {
          throw new Error(`文書を取得できませんでした: ${response.status}`);
        }

        return response.text();
      })
      .then((markdown) => {
        setOpenDocument({ name: item.name, content: parseMarkdownDocument(markdown) });
      })
      .catch(() => setOpenDocument(null));
  }, []);

  return (
    <div className="min-h-dvh">
      <header className="border-border border-b">
        <div className="mx-auto flex max-w-7xl flex-col gap-4 px-6 py-6">
          <div>
            <h1 className="font-semibold text-2xl">{docs.title}</h1>
            <p className="text-muted-foreground text-sm">{docs.subtitle}</p>
          </div>

          <div className="flex flex-wrap items-center gap-4">
            <SearchFieldClient
              className="max-w-md flex-1"
              label="ドキュメントを検索"
              onSearch={setQuery}
            />
            <ToggleGroupNative aria-label="表示言語">
              <ToggleGroupNativeItem
                checked={lang === "EN"}
                name="lang"
                onChange={selectEnglish}
                value="EN"
              >
                EN
              </ToggleGroupNativeItem>
              <ToggleGroupNativeItem
                checked={lang === "JA"}
                name="lang"
                onChange={selectJapanese}
                value="JA"
              >
                JA
              </ToggleGroupNativeItem>
            </ToggleGroupNative>
          </div>
        </div>
      </header>

      <div className="mx-auto flex max-w-7xl flex-col gap-8 px-6 py-8 md:flex-row">
        <aside className="md:w-64 md:shrink-0">
          <PortalSidebar
            activeGroupSlug={activeGroupSlug}
            groups={visibleGroups}
            referenceLinks={docs.referenceLinks}
          />
        </aside>

        <main className="min-w-0 flex-1">
          {results ? (
            <section aria-labelledby={searchResultsHeadingId} className="flex flex-col gap-4">
              <h2 className="font-semibold text-xl" id={searchResultsHeadingId}>
                検索結果 {results.length} 件
              </h2>
              {results.length === 0 ? (
                <p className="text-muted-foreground">一致する項目がありません。</p>
              ) : (
                <PortalCardGrid items={results} onOpenDocument={onOpenDocument} />
              )}
            </section>
          ) : activeGroup ? (
            <div className="flex flex-col gap-8">
              <h2 className="font-semibold text-xl">{activeGroup.title}</h2>
              {activeGroup.sections.map((section) => (
                <section
                  aria-labelledby={`section-${section.slug}-heading`}
                  className="flex flex-col gap-4"
                  id={`section-${section.slug}`}
                  key={section.slug}
                >
                  <h3 className="font-medium text-lg" id={`section-${section.slug}-heading`}>
                    {section.title}
                  </h3>
                  <PortalCardGrid items={section.items} onOpenDocument={onOpenDocument} />
                  {(section.subgroups ?? []).map((subgroup) => (
                    <div className="flex flex-col gap-3" key={subgroup.title}>
                      <h4 className="font-medium text-muted-foreground text-sm">
                        {subgroup.title}
                      </h4>
                      <PortalCardGrid items={subgroup.items} onOpenDocument={onOpenDocument} />
                    </div>
                  ))}
                </section>
              ))}
            </div>
          ) : (
            <p className="text-muted-foreground">表示できる項目がありません。</p>
          )}
        </main>
      </div>

      {/* trigger を持たないため、開くのはカード側からだけ。閉じる要求だけがここへ来る。 */}
      <Dialog onOpenChange={closeDocument} open={openDocument !== null}>
        <DialogContent className="max-h-[85dvh] overflow-y-auto sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>{openDocument?.name ?? ""}</DialogTitle>
          </DialogHeader>
          {openDocument?.content ? (
            // `DialogContent` は grid で、track は既定で中身の最小幅まで広がる。横に長い
            // コードブロックや表がそのまま面をはみ出すため、縮められる状態にしておく。
            <DocumentContent className="min-w-0" content={openDocument.content} />
          ) : (
            <p className="text-muted-foreground">読み込んでいます...</p>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
