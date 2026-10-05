"use client";

import dynamic from "next/dynamic";

const DotLottieReact = dynamic(
  () => import("@lottiefiles/dotlottie-react").then((mod) => mod.DotLottieReact),
  { ssr: false },
);

interface HeroAnimationProps {
  src: string;
}

export default function HeroAnimation({ src }: HeroAnimationProps) {
  return (
    <div className="w-full flex items-center justify-center">
      <div className="w-full max-w-105 sm:max-w-145 lg:max-w-180 aspect-square drop-shadow-sm select-none pointer-events-none">
        <DotLottieReact
          src={src}
          autoplay
          loop
          className="w-full h-full"
        />
      </div>
    </div>
  );
}