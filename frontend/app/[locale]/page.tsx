// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

"use client";

import { useCallback } from "react";
import dynamic from "next/dynamic";
import { useTranslations } from "next-intl";
import { useRouter } from "next/navigation";
import confetti from "canvas-confetti";
import {
  FileUp,
  Users,
  BarChart3,
  Shield,
  Clock,
  Globe,
  Github,
  Mail,
} from "lucide-react";
import { ThemeToggle } from "@/components/ui/theme-toggle";
import { LocaleToggle } from "@/components/ui/locale-toggle";
import { Footer } from "@/components/ui/modem-animated-footer";

const PortfolioPage = dynamic(
  () =>
    import("@/components/ui/starfall-portfolio-landing").then(
      (mod) => mod.PortfolioPage
    ),
  { ssr: false }
);

export default function LandingPage() {
  const t = useTranslations("landing");
  const router = useRouter();

  const words = t.raw("confetti") as string[];

  const handleBrandClick = useCallback(() => {
    const colors = ["#a786ff", "#fd8bbc", "#eca184", "#f8deb1"];

    // Word shapes in bright colors visible on any background
    const brightColors = ["#ff6b6b", "#ffd93d", "#6bcb77", "#4d96ff", "#ff6bd6", "#ffa94d"];
    const wordShapes = words.map((w, i) =>
      confetti.shapeFromText({ text: w, scalar: 3, color: brightColors[i % brightColors.length] })
    );

    // Custom SVG shapes instead of ugly emojis
    const star = confetti.shapeFromPath({ path: "M5 0 L6.2 3.8 L10 3.8 L7 6.2 L8.2 10 L5 7.6 L1.8 10 L3 6.2 L0 3.8 L3.8 3.8 Z" });
    const diamond = confetti.shapeFromPath({ path: "M5 0 L10 5 L5 10 L0 5 Z" });
    const triangle = confetti.shapeFromPath({ path: "M5 0 L10 10 L0 10 Z" });

    // 1) Stars burst — gold/yellow
    const starDefaults = {
      spread: 360, ticks: 60, gravity: 0, decay: 0.94, startVelocity: 30,
      colors: ["#FFE400", "#FFBD00", "#E89400", "#FFCA6C", "#FDFFB8"],
    };
    const shootStars = () => {
      confetti({ ...starDefaults, particleCount: 40, scalar: 1.2, shapes: ["star"] });
      confetti({ ...starDefaults, particleCount: 10, scalar: 0.75, shapes: ["circle"] });
    };
    setTimeout(shootStars, 0);
    setTimeout(shootStars, 100);
    setTimeout(shootStars, 200);

    // 2) Side cannons — colored confetti + SVG shapes
    const end = Date.now() + 800;
    const frame = () => {
      if (Date.now() > end) return;
      confetti({ particleCount: 3, angle: 60, spread: 55, startVelocity: 50, origin: { x: 0, y: 0.5 }, colors, shapes: [star, diamond, triangle, "circle"], scalar: 2 });
      confetti({ particleCount: 3, angle: 120, spread: 55, startVelocity: 50, origin: { x: 1, y: 0.5 }, colors, shapes: [star, diamond, triangle, "circle"], scalar: 2 });
      requestAnimationFrame(frame);
    };
    frame();

    // 3) Flying translated words in bright colors
    confetti({
      particleCount: words.length,
      angle: 90,
      spread: 160,
      startVelocity: 25,
      origin: { x: 0.5, y: 0.9 },
      shapes: wordShapes,
      scalar: 3,
      gravity: 0.4,
      ticks: 300,
      flat: true,
    });
  }, [words]);

  return (
    <PortfolioPage
      logo={{
        initials: "EP",
        name: "EstimatePro",
      }}
      navLinks={[
        { label: t("nav.features"), href: "#features" },
        { label: t("nav.stats"), href: "#stats" },
      ]}
      navExtra={<><LocaleToggle /><ThemeToggle /></>}
      resume={{
        label: t("nav.login"),
        onClick: () => router.push("/login"),
      }}
      hero={{
        titleLine1: t("hero.line1"),
        titleLine2Gradient: t("hero.line2"),
        subtitle: t("hero.subtitle"),
      }}
      ctaButtons={{
        primary: {
          label: t("cta"),
          onClick: () => router.push("/register"),
        },
        secondary: {
          label: t("ctaSecondary"),
          onClick: () => {
            document
              .getElementById("features")
              ?.scrollIntoView({ behavior: "smooth" });
          },
        },
      }}
      projects={[
        {
          title: t("features.upload"),
          description: t("features.uploadDesc"),
          tags: ["PDF", "DOCX", "XLSX", "MD"],
          imageContent: (
            <FileUp className="w-12 h-12 text-muted-foreground/50" />
          ),
        },
        {
          title: t("features.collaborate"),
          description: t("features.collaborateDesc"),
          tags: ["PM", "Tech Lead", "Developer"],
          imageContent: (
            <Users className="w-12 h-12 text-muted-foreground/50" />
          ),
        },
        {
          title: t("features.aggregate"),
          description: t("features.aggregateDesc"),
          tags: ["Avg", "Min", "Max", "Sum"],
          imageContent: (
            <BarChart3 className="w-12 h-12 text-muted-foreground/50" />
          ),
        },
        {
          title: t("features.roles"),
          description: t("features.rolesDesc"),
          tags: ["Admin", "PM", "Dev", "Observer"],
          imageContent: (
            <Shield className="w-12 h-12 text-muted-foreground/50" />
          ),
        },
        {
          title: t("features.versions"),
          description: t("features.versionsDesc"),
          tags: ["Timeline", "Diff", "History"],
          imageContent: (
            <Clock className="w-12 h-12 text-muted-foreground/50" />
          ),
        },
        {
          title: t("features.i18n"),
          description: t("features.i18nDesc"),
          tags: ["RU", "EN"],
          imageContent: (
            <Globe className="w-12 h-12 text-muted-foreground/50" />
          ),
        },
      ]}
      stats={[
        { value: t("stats.formatsValue"), label: t("stats.formatsLabel") },
        { value: t("stats.rolesValue"), label: t("stats.rolesLabel") },
        { value: t("stats.realtimeValue"), label: t("stats.realtimeLabel") },
      ]}
      showAnimatedBackground={true}
    >
      <Footer
        brandName="EstimatePro"
        brandDescription={t("footer.description")}
        onBrandClick={handleBrandClick}
        brandIcon={
          <span className="text-2xl sm:text-3xl md:text-4xl font-bold text-primary-foreground">
            EP
          </span>
        }
        socialLinks={[
          {
            icon: <Github className="w-full h-full" />,
            href: "https://github.com",
            label: "GitHub",
          },
          {
            icon: <Mail className="w-full h-full" />,
            href: "mailto:hello@estimatepro.io",
            label: "Email",
          },
        ]}
        navLinks={[
          { label: t("footer.features"), href: "#features" },
          { label: t("footer.pricing"), href: "#" },
          { label: t("footer.docs"), href: "#" },
          { label: t("footer.support"), href: "#" },
          { label: t("footer.privacy"), href: "#" },
          { label: t("footer.terms"), href: "#" },
        ]}
      />
    </PortfolioPage>
  );
}
