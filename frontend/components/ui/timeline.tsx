"use client";

import React, { useEffect, useRef, useState } from "react";

export interface TimelineEntry {
  title: string;
  content: React.ReactNode;
  completed?: boolean;
}

function getDotColor(index: number, total: number) {
  const t = total <= 1 ? 0 : index / (total - 1);
  // blue (#3b82f6) → indigo (#6366f1) → teal (#14b8a6) → emerald (#10b981)
  let r: number, g: number, b: number;
  if (t < 0.33) {
    const s = t / 0.33;
    r = Math.round(0x3b + (0x63 - 0x3b) * s);
    g = Math.round(0x82 + (0x66 - 0x82) * s);
    b = Math.round(0xf6 + (0xf1 - 0xf6) * s);
  } else if (t < 0.66) {
    const s = (t - 0.33) / 0.33;
    r = Math.round(0x63 + (0x14 - 0x63) * s);
    g = Math.round(0x66 + (0xb8 - 0x66) * s);
    b = Math.round(0xf1 + (0xa6 - 0xf1) * s);
  } else {
    const s = (t - 0.66) / 0.34;
    r = Math.round(0x14 + (0x10 - 0x14) * s);
    g = Math.round(0xb8 + (0xb9 - 0xb8) * s);
    b = Math.round(0xa6 + (0x81 - 0xa6) * s);
  }
  return `rgb(${r}, ${g}, ${b})`;
}

interface Segment {
  top: number;
  height: number;
  colorFrom: string;
  colorTo: string;
  isGray: boolean;
}

export function Timeline({ data }: { data: TimelineEntry[] }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const dotRefs = useRef<(HTMLDivElement | null)[]>([]);
  const [segments, setSegments] = useState<Segment[]>([]);

  useEffect(() => {
    const dots = dotRefs.current.filter(Boolean) as HTMLDivElement[];
    if (dots.length < 2 || !containerRef.current) return;

    const containerRect = containerRef.current.getBoundingClientRect();
    const getCenter = (el: HTMLDivElement) =>
      el.getBoundingClientRect().top - containerRect.top + el.getBoundingClientRect().height / 2;

    const newSegments: Segment[] = [];

    for (let i = 0; i < dots.length - 1; i++) {
      const topCenter = getCenter(dots[i]);
      const bottomCenter = getCenter(dots[i + 1]);
      const topCompleted = data[i]?.completed ?? false;
      const bottomCompleted = data[i + 1]?.completed ?? false;
      const bothCompleted = topCompleted && bottomCompleted;

      newSegments.push({
        top: topCenter,
        height: bottomCenter - topCenter,
        colorFrom: getDotColor(i, data.length),
        colorTo: getDotColor(i + 1, data.length),
        isGray: !bothCompleted,
      });
    }

    // eslint-disable-next-line react-hooks/set-state-in-effect -- layout measurement requires setState after DOM read
    setSegments(newSegments);
  }, [data]);

  return (
    <div className="w-full font-sans" ref={containerRef}>
      <div className="relative max-w-7xl mx-auto pb-10">
        {data.map((item, index) => {
          const dotColor = item.completed
            ? getDotColor(index, data.length)
            : undefined;

          return (
            <div
              key={index}
              className="flex justify-start pt-6 md:pt-16 md:gap-10"
            >
              <div className="sticky flex flex-col md:flex-row z-40 items-center top-40 self-start max-w-xs lg:max-w-sm md:w-full">
                <div
                  ref={(el) => { dotRefs.current[index] = el; }}
                  className="h-10 absolute left-3 md:left-3 w-10 rounded-full bg-background flex items-center justify-center"
                >
                  {item.completed ? (
                    <div
                      className="h-4 w-4 rounded-full flex items-center justify-center p-2"
                      style={{ backgroundColor: dotColor }}
                    >
                      <svg className="h-3 w-3 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
                        <polyline points="20 6 9 17 4 12" />
                      </svg>
                    </div>
                  ) : (
                    <div className="h-4 w-4 rounded-full border border-border p-2" style={{ backgroundColor: "var(--muted)" }} />
                  )}
                </div>
                <h3
                  className={`hidden md:block text-xl md:pl-20 md:text-3xl font-bold ${item.completed ? "text-foreground" : "text-muted-foreground"}`}
                >
                  {item.title}
                </h3>
              </div>

              <div className="relative pl-20 pr-4 md:pl-4 w-full">
                <h3 className={`md:hidden block text-xl mb-2 text-left font-bold ${item.completed ? "text-foreground" : "text-muted-foreground"}`}>
                  {item.title}
                </h3>
                {item.content}
              </div>
            </div>
          );
        })}

        {/* Line segments between each pair of dots */}
        {segments.map((seg, i) => (
          <div
            key={i}
            style={{
              top: seg.top,
              height: seg.height,
              background: seg.isGray
                ? "var(--muted)"
                : `linear-gradient(to bottom, ${seg.colorFrom}, ${seg.colorTo})`,
            }}
            className="absolute md:left-8 left-8 w-[2px]"
          />
        ))}
      </div>
    </div>
  );
}
