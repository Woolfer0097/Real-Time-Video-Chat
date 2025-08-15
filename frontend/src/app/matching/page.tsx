"use client";

import Image from "next/image";
import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";

type Language = "en" | "ru";

const TEXT: Record<
  Language,
  {
    title: string;
    subtitle: string;
    randomTitle: string;
    randomSubtitle: string;
    randomButton: string;
    orText: string;
    cefrLabel: string;
    cefrLevels: readonly string[];
    ageLabel: string;
    ageRange: string;
    genderLabel: string;
    genderOptions: readonly string[];
    interestsLabel: string;
    interestsOptions: readonly string[];
    topicsLabel: string;
    topicsOptions: readonly string[];
    dataNote: string;
    startMatchmaking: string;
    languageName: string;
  }
> = {
  en: {
    title: "Find your conversation partner",
    subtitle: "Choose how you want to find your English practice partner.",
    randomTitle: "Quick Start",
    randomSubtitle: "Start practicing immediately with any available partner",
    randomButton: "Search randomly",
    orText: "or",
    cefrLabel: "Preferred CEFR level",
    cefrLevels: [
      "Any level",
      "Beginner",
      "Elementary", 
      "Intermediate",
      "Upper-intermediate",
      "Advanced",
      "Proficient",
    ],
    ageLabel: "Preferred age",
    ageRange: "Age range",
    genderLabel: "Preferred gender",
    genderOptions: ["Any gender", "Male", "Female"],
    interestsLabel: "Shared interests",
    interestsOptions: [
      "Any interests",
      "Playing games",
      "Reading books",
      "Watching movies",
      "Music",
      "Cooking",
      "Traveling",
      "Fitness",
      "Photography",
      "Coding",
      "Art & Design",
      "Gardening",
      "DIY & Crafts",
    ],
    topicsLabel: "Conversation topics",
    topicsOptions: [
      "Any topics",
      "Daily life",
      "Hobbies & interests",
      "Travel & culture",
      "Technology",
      "Movies & books",
      "Food & cooking",
      "Sports & fitness",
      "Work & career",
      "Education",
      "Current events",
    ],
    dataNote: "We'll use this to find the best matches for you.",
    startMatchmaking: "Start matchmaking",
    languageName: "English",
  },
  ru: {
    title: "Найди собеседника",
    subtitle: "Выбери, как ты хочешь найти партнера для практики английского.",
    randomTitle: "Быстрый старт",
    randomSubtitle: "Начни практиковаться сразу с любым доступным партнером",
    randomButton: "Искать рандомно",
    orText: "или",
    cefrLabel: "Предпочитаемый уровень CEFR",
    cefrLevels: [
      "Любой уровень",
      "Начальный",
      "Элементарный",
      "Средний",
      "Выше среднего",
      "Продвинутый",
      "Профессиональный",
    ],
    ageLabel: "Предпочитаемый возраст",
    ageRange: "Возрастной диапазон",
    genderLabel: "Предпочитаемый пол",
    genderOptions: ["Любой пол", "Мужчина", "Женщина"],
    interestsLabel: "Общие интересы",
    interestsOptions: [
      "Любые интересы",
      "Игры",
      "Чтение",
      "Просмотр фильмов",
      "Музыка",
      "Кулинария",
      "Путешествия",
      "Фитнес",
      "Фотография",
      "Программирование",
      "Искусство и дизайн",
      "Садоводство",
      "Рукоделие",
    ],
    topicsLabel: "Темы для разговора",
    topicsOptions: [
      "Любые темы",
      "Повседневная жизнь",
      "Хобби и интересы",
      "Путешествия и культура",
      "Технологии",
      "Фильмы и книги",
      "Еда и кулинария",
      "Спорт и фитнес",
      "Работа и карьера",
      "Образование",
      "Текущие события",
    ],
    dataNote: "Мы используем это, чтобы найти лучших собеседников для вас",
    startMatchmaking: "Начать поиск",
    languageName: "Русский",
  },
};

export default function MatchingPage() {
  const router = useRouter();
  const [language, setLanguage] = useState<Language>("en");
  const [showQuestionnaire, setShowQuestionnaire] = useState(false);
  const [cefrLevel, setCefrLevel] = useState<string>("");
  const [ageRange, setAgeRange] = useState<string>("");
  const [gender, setGender] = useState<string>("");
  const [interests, setInterests] = useState<Set<string>>(new Set());
  const [topics, setTopics] = useState<Set<string>>(new Set());
  const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8000";

  // Load and persist language selection
  useEffect(() => {
    try {
      const stored = localStorage.getItem("lang");
      if (stored === "en" || stored === "ru") setLanguage(stored);
    } catch {}
  }, []);
  useEffect(() => {
    try {
      localStorage.setItem("lang", language);
    } catch {}
  }, [language]);

  const t = TEXT[language];

  const isValid = useMemo(() => {
    return cefrLevel !== "" && ageRange !== "" && gender !== "";
  }, [cefrLevel, ageRange, gender]);

  const toggleInterest = (interest: string) => {
    setInterests((prev) => {
      const next = new Set(prev);
      if (next.has(interest)) next.delete(interest);
      else next.add(interest);
      return next;
    });
  };

  const toggleTopic = (topic: string) => {
    setTopics((prev) => {
      const next = new Set(prev);
      if (next.has(topic)) next.delete(topic);
      else next.add(topic);
      return next;
    });
  };

  const handleSearchRandom = async () => {
    try {
      const userId = typeof window !== "undefined" ? localStorage.getItem("user_id") : null;
      if (!userId) {
        alert("Please create your profile first");
        router.push("/");
        return;
      }
      
      // Redirect to waiting page instead of immediate matching
      router.push("/waiting");
    } catch (e) {
      console.error(e);
      alert("Failed to start matchmaking. Try again.");
    }
  };

  const handleStartMatchmaking = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const userId = typeof window !== "undefined" ? localStorage.getItem("user_id") : null;
      if (!userId) {
        alert("Please create your profile first");
        router.push("/");
        return;
      }
      // Optionally update user topics for similarity
      const update = {
        id: userId,
        topics: Array.from(topics),
        cefr_level: cefrLevel !== "Any level" ? cefrLevel : "",
        gender: gender !== "Any gender" ? gender : "",
        // rough age bucket could be inferred backend-side
      };
      await fetch(`${API_BASE}/api/users`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(update),
      });

      // Redirect to waiting page instead of immediate matching
      router.push("/waiting");
    } catch (err) {
      console.error(err);
      alert("Failed to start matchmaking. Try again.");
    }
  };

  return (
    <div className="min-h-screen w-full font-body">
      <div className="mx-auto max-w-3xl p-6 sm:p-10">
        <header className="mb-8 flex items-start justify-between gap-4">
          <h1
            className="font-heading text-2xl sm:text-3xl font-bold"
            style={{
              backgroundImage: "linear-gradient(to right, var(--orange), var(--light-orange))",
              WebkitBackgroundClip: "text",
              color: "transparent",
            }}
          >
            {t.title}
          </h1>
          <div className="flex flex-col items-end gap-2">
            <button
              type="button"
              aria-label="Switch language"
              className="flag-switcher"
              onClick={() => setLanguage((prev) => (prev === "en" ? "ru" : "en"))}
            >
              {/* Active Flag */}
              <div className={`flag-card ${language === "en" ? "flag-active rotate-swap" : "flag-inactive"}`}>
                <Image src="/flags/en.svg" alt="English" width={160} height={112} />
              </div>
              {/* Inactive Flag at 45deg up-right behind */}
              <div className={`flag-card ${language === "ru" ? "flag-active rotate-swap" : "flag-inactive"}`}>
                <Image src="/flags/ru.svg" alt="Русский" width={160} height={112} />
              </div>
            </button>
            <span className="text-xs text-[--color-muted] uppercase tracking-wide">
              {t.languageName}
            </span>
          </div>
        </header>
        <p className="text-[--color-muted] mb-6">{t.subtitle}</p>

        {!showQuestionnaire ? (
          // Random search option
          <div className="rounded-2xl p-6 sm:p-8 border border-white/10 bg-[--color-card] shadow-xl shadow-black/30">
            <div className="text-center space-y-6">
              <div className="space-y-2">
                <h2 className="font-heading text-xl font-semibold">{t.randomTitle}</h2>
                <p className="text-[--color-muted]">{t.randomSubtitle}</p>
              </div>
              
              <button
                onClick={handleSearchRandom}
                className="button-primary w-full sm:w-auto px-8 py-3 text-lg"
              >
                {t.randomButton}
              </button>
              
              <div className="flex items-center justify-center gap-4">
                <div className="flex-1 h-px bg-white/10"></div>
                <span className="text-sm text-[--color-muted] px-4">{t.orText}</span>
                <div className="flex-1 h-px bg-white/10"></div>
              </div>
              
              <button
                onClick={() => setShowQuestionnaire(true)}
                className="chip hover:bg-white/10 transition-colors"
              >
                {t.startMatchmaking}
              </button>
            </div>
          </div>
        ) : (
          // Questionnaire
          <div className="rounded-2xl p-6 sm:p-8 border border-white/10 bg-[--color-card] shadow-xl shadow-black/30">
            <form className="space-y-6" onSubmit={handleStartMatchmaking}>
              {/* CEFR Level */}
              <div className="space-y-2">
                <label className="block text-sm font-medium font-heading">{t.cefrLabel}</label>
                <div className="flex flex-wrap gap-2">
                  {t.cefrLevels.map((level) => {
                    const selected = cefrLevel === level;
                    return (
                      <button
                        key={level}
                        type="button"
                        onClick={() => setCefrLevel(level)}
                        className={`chip ${selected ? "chip-selected" : ""}`}
                      >
                        {level}
                      </button>
                    );
                  })}
                </div>
              </div>

              {/* Age Range */}
              <div className="space-y-2">
                <label className="block text-sm font-medium font-heading">{t.ageLabel}</label>
                <div className="flex flex-wrap gap-2">
                  {["Any age", "18-25", "26-35", "36-45", "46-55", "55+"].map((range) => {
                    const selected = ageRange === range;
                    return (
                      <button
                        key={range}
                        type="button"
                        onClick={() => setAgeRange(range)}
                        className={`chip ${selected ? "chip-selected" : ""}`}
                      >
                        {range}
                      </button>
                    );
                  })}
                </div>
              </div>

              {/* Gender */}
              <div className="space-y-2">
                <label className="block text-sm font-medium font-heading">{t.genderLabel}</label>
                <div className="flex flex-wrap gap-2">
                  {t.genderOptions.map((g) => {
                    const selected = gender === g;
                    return (
                      <button
                        key={g}
                        type="button"
                        onClick={() => setGender(g)}
                        className={`chip ${selected ? "chip-selected" : ""}`}
                      >
                        {g}
                      </button>
                    );
                  })}
                </div>
              </div>

              {/* Interests */}
              <div className="space-y-2">
                <label className="block text-sm font-medium font-heading">{t.interestsLabel}</label>
                <div className="flex flex-wrap gap-2">
                  {t.interestsOptions.map((interest) => {
                    const selected = interests.has(interest);
                    return (
                      <button
                        key={interest}
                        type="button"
                        onClick={() => toggleInterest(interest)}
                        className={`chip ${selected ? "chip-selected" : ""}`}
                      >
                        {interest}
                      </button>
                    );
                  })}
                </div>
              </div>

              {/* Topics */}
              <div className="space-y-2">
                <label className="block text-sm font-medium font-heading">{t.topicsLabel}</label>
                <div className="flex flex-wrap gap-2">
                  {t.topicsOptions.map((topic) => {
                    const selected = topics.has(topic);
                    return (
                      <button
                        key={topic}
                        type="button"
                        onClick={() => toggleTopic(topic)}
                        className={`chip ${selected ? "chip-selected" : ""}`}
                      >
                        {topic}
                      </button>
                    );
                  })}
                </div>
              </div>

              <div className="pt-2 flex items-center justify-between gap-4">
                <p className="text-xs text-[--color-muted]">{t.dataNote}</p>
                <button disabled={!isValid} className="button-primary disabled:opacity-50" type="submit">
                  {t.startMatchmaking}
                </button>
              </div>
            </form>
          </div>
        )}
      </div>
    </div>
  );
}
