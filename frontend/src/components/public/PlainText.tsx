import { toPlainTextModel } from "@/lib/plain-text";
export function PlainText({ value }: { value: string }) { return <div className="plain-text">{toPlainTextModel(value).map((paragraph, i) => <p key={i}>{paragraph.map((token, j) => token.kind === "link" ? <a key={j} href={token.href} rel="noopener noreferrer">{token.value}</a> : <span key={j}>{token.value}</span>)}</p>)}</div>; }
