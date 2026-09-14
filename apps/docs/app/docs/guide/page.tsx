import { redirect } from "next/navigation";

export default function DocsRootPage() {
  redirect("/docs/guide/01-lifecycle");
}