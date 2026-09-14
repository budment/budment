import { redirect } from "next/navigation";

export default function DocsRootPage() {
  redirect("/docs/reference/cli");
}