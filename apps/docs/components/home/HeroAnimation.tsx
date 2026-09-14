"use client";

import dynamic from "next/dynamic";

const DotLottiePlayer = dynamic(
  () => import("@dotlottie/react-player").then((mod) => mod.DotLottiePlayer),
  { ssr: false },
);

interface HeroAnimationProps {
  src: string;
}

export default function HeroAnimation({ src }: HeroAnimationProps) {
  return (
    <div className="w-full flex items-center justify-center">
      <DotLottiePlayer
        src={src}
        autoplay
        loop
        className="w-full max-w-95 sm:max-w-120 lg:max-w-140 h-auto drop-shadow-sm select-none pointer-events-none"
      />
    </div>
  );
}
