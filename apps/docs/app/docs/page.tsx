import { redirect } from "next/navigation";

export default function DocsRootPage() {
  redirect("/docs/getting-started/01-introduction");
}