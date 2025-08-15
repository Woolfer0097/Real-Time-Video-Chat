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
    cefrLabel: string;
    cefrLevels: readonly string[];
    ageLabel: string;
    agePlaceholder: string;
    genderLabel: string;
    genderOptions: readonly string[];
    interestsLabel: string;
    interestsOptions: readonly string[];
    dataNote: string;
    continue: string;
    languageName: string;
    userNameLabel: string;
    userNamePlaceholder: string;
    userNameSubmit: string;
    userNameError: string;
    userNameThanks: string;
  }
> = {
  en: {
    title: "Tell us about yourself",
    subtitle: "We will use this to personalize your experience.",
    cefrLabel: "CEFR level",
    cefrLevels: [
      "Beginner",
      "Elementary",
      "Intermediate",
      "Upper-intermediate",
      "Advanced",
      "Proficient",
    ],
    ageLabel: "Age",
    agePlaceholder: "e.g., 25",
    genderLabel: "Gender",
    genderOptions: ["Male", "Female"],
    interestsLabel: "Interests",
    interestsOptions: [
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
    dataNote: "Your data stays on this device unless you proceed.",
    continue: "Continue",
    languageName: "English",
    userNameLabel: "Your Name",
    userNamePlaceholder: "Enter your name",
    userNameSubmit: "OK",
    userNameError: "Name is required.",
    userNameThanks: "Thank you!",
  },
  ru: {
    title: "Расскажите о себе",
    subtitle: "Мы используем это, чтобы персонализировать ваш опыт.",
    cefrLabel: "Уровень CEFR",
    cefrLevels: [
      "Начальный",
      "Элементарный",
      "Средний",
      "Выше среднего",
      "Продвинутый",
      "Профессиональный",
    ],
    ageLabel: "Возраст",
    agePlaceholder: "например, 25",
    genderLabel: "Пол",
    genderOptions: [
      "Мужчина",
      "Женщина"
    ],
    interestsLabel: "Интересы",
    interestsOptions: [
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
    dataNote: "Ваши данные остаются на этом устройстве, пока вы не продолжите.",
    continue: "Продолжить",
    languageName: "Русский",
    userNameLabel: "Ваше имя",
    userNamePlaceholder: "Введите ваше имя",
    userNameSubmit: "ОК",
    userNameError: "Имя обязательно.",
    userNameThanks: "Спасибо!",
  },
};

export default function Home() {
  const router = useRouter();
  const [language, setLanguage] = useState<Language>("en");
  const [level, setLevel] = useState<string>("");
  const [age, setAge] = useState<string>("");
  const [gender, setGender] = useState<string>("");
  const [interests, setInterests] = useState<Set<string>>(new Set());
  const [userName, setUserName] = useState<string>("");
  const [userNameError, setUserNameError] = useState<boolean>(false);
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
    const valid = (
      userName.trim() !== "" &&
      level !== "" &&
      age !== "" &&
      gender !== "" &&
      interests.size > 0
    );
    console.log("Form validation:", {
      userName: userName.trim(),
      level,
      age,
      gender,
      interestsSize: interests.size,
      isValid: valid
    });
    return valid;
  }, [userName, level, age, gender, interests]);

  const toggleInterest = (interest: string) => {
    setInterests((prev) => {
      const next = new Set(prev);
      if (next.has(interest)) next.delete(interest);
      else next.add(interest);
      return next;
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    console.log("Form submitted");
    console.log("API_BASE:", API_BASE);
    
    const payload = {
      name: userName,
      language,
      cefr_level: level,
      age: Number(age),
      gender,
      interests: Array.from(interests),
    } as const;
    
    console.log("Payload:", payload);

    try {
      console.log("Making API request to:", `${API_BASE}/api/users`);
      const res = await fetch(`${API_BASE}/api/users`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      console.log("Response status:", res.status);
      console.log("Response ok:", res.ok);
      
      if (!res.ok) throw new Error("Failed to create user");
      const data = await res.json();
      console.log("Response data:", data);
      
      try {
        localStorage.setItem("user_id", data.id);
        console.log("User ID saved to localStorage:", data.id);
      } catch {}
      console.log("Redirecting to /matching");
      router.push("/matching");
    } catch (err) {
      console.error("Error in handleSubmit:", err);
      alert("Failed to save profile. Please try again.");
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
          </div>
          {/* Name moved into the main form below */}
        </header>
        <p className="text-[--color-muted] mb-6">{t.subtitle}</p>

        <div className="rounded-2xl p-6 sm:p-8 border border-white/10 bg-[--color-card] shadow-xl shadow-black/30">
          <form className="space-y-6" onSubmit={handleSubmit} autoComplete="off">
            {/* Name */}
            <div className="space-y-2">
              <label htmlFor="userName" className="block text-sm font-medium font-heading">{t.userNameLabel}</label>
              <input
                id="userName"
                type="text"
                className="input"
                value={userName}
                onChange={(e) => {
                  setUserName(e.target.value);
                  setUserNameError(false);
                }}
                placeholder={t.userNamePlaceholder}
                required
                minLength={1}
                maxLength={64}
                style={{ minWidth: 180 }}
              />
              {userNameError && (
                <p className="text-red-500 text-xs mt-1">{t.userNameError}</p>
              )}
            </div>
            {/* CEFR Level */}
            <div className="space-y-2">
              <label className="block text-sm font-medium font-heading">{t.cefrLabel}</label>
              <div className="flex flex-wrap gap-2">
                {t.cefrLevels.map((lvl) => {
                  const selected = level === lvl;
                  return (
                    <button
                      key={lvl}
                      type="button"
                      onClick={() => setLevel(lvl)}
                      className={`chip ${selected ? "chip-selected" : ""}`}
                    >
                      {lvl}
                    </button>
                  );
                })}
              </div>
            </div>

            {/* Age */}
            <div className="space-y-2">
              <label htmlFor="age" className="block text-sm font-medium font-heading">{t.ageLabel}</label>
              <input
                id="age"
                inputMode="numeric"
                pattern="[0-9]*"
                placeholder={t.agePlaceholder}
                className="input"
                value={age}
                onChange={(e) => setAge(e.target.value.replace(/[^0-9]/g, ""))}
              />
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

            <div className="pt-2 flex items-center justify-between gap-4">
              <p className="text-xs text-[--color-muted]">{t.dataNote}</p>
               <button 
                disabled={!isValid} 
                className="button-primary disabled:opacity-50" 
                type="submit"
                onClick={() => console.log("Continue button clicked, isValid:", isValid)}
              >
                {t.continue}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}
