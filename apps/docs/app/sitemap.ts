import type { MetadataRoute } from "next";
import { generateStaticParams } from "./docs/[...slug]/page";

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const docsParams = await generateStaticParams();
  const docUrls = docsParams
    .filter((p) => p.slug.length > 0)
    .map((p) => ({
      url: `https://budment.com/docs/${p.slug.join("/")}`,
      lastModified: new Date(),
      changeFrequency: "weekly" as const,
      priority: 0.8,
    }));

  return [
    {
      url: "https://budment.com",
      lastModified: new Date(),
      changeFrequency: "monthly",
      priority: 1.0,
    },
    ...docUrls,
  ];
}