import { createContext, useContext, useState, useEffect, ReactNode } from 'react';

type Theme = 'light' | 'dark';

interface ThemeContent {
    theme: Theme;
    setTheme: (theme: Theme) => void;
}

const getInitialTheme = (): Theme => {
    const value = localStorage.getItem('theme');
    if (value === 'light') return 'light';
    return 'dark';
};

const ThemeContext = createContext<ThemeContent | null>(null);

export const ThemeProvider = ({ children }: { children: ReactNode }) => {
    const [theme, setTheme] = useState<Theme>(() => getInitialTheme());

    useEffect(() => {
        localStorage.setItem('theme', theme);
        document.documentElement.setAttribute('data-theme', theme);
    }, [theme]);

    return (
        <ThemeContext.Provider value={{ theme, setTheme }}>
            {children}
        </ThemeContext.Provider>
    );
};

export const useTheme = () => {
    const context = useContext(ThemeContext);
    if (!context) {
        throw new Error('useTheme должен использоваться только внутри ThemeProvider');
    }
    return context;
};