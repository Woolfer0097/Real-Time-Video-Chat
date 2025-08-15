import type { Metadata } from "next";
import localFont from "next/font/local";
import "./globals.css";

const inter = localFont({
  src: [
    {
      path: "../../public/fonts/Inter/Inter-Regular.otf",
      weight: "400",
      style: "normal",
    },
    {
      path: "../../public/fonts/Inter/Inter-Medium.otf",
      weight: "500",
      style: "normal",
    },
    {
      path: "../../public/fonts/Inter/Inter-SemiBold.otf",
      weight: "600",
      style: "normal",
    },
    {
      path: "../../public/fonts/Inter/Inter-Bold.otf",
      weight: "700",
      style: "normal",
    },
  ],
  variable: "--font-inter",
  display: "swap",
});

const fraunces = localFont({
  src: [
    {
      path: "../../public/fonts/Fraunces/Fraunces.ttf",
      weight: "400",
      style: "normal",
    },
    {
      path: "../../public/fonts/Fraunces/Fraunces-Italic.ttf",
      weight: "400",
      style: "italic",
    },
  ],
  variable: "--font-fraunces",
  display: "swap",
});

export const metadata: Metadata = {
  title: "User Onboarding",
  description: "Collect CEFR level, age, gender, and interests",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className={`${inter.variable} ${fraunces.variable} antialiased bg-[--color-background] text-[--color-foreground]`}>
        <div className="min-h-screen bg-[--color-background]">
          {children}
        </div>
      </body>
    </html>
  );
}
