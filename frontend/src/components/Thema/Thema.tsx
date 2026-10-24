import { useEffect, useRef, useState } from 'react';
import { useTheme } from '../../context/ThemeContext';
import moon from '../../image/moon.png';
import sun from '../../image/sun.png';
import './Thema.css';

const ThemeSwap = () => {
    const { theme, setTheme } = useTheme();
    const [overlay, setOverlay] = useState<
        null | { x: number; y: number; radius: number; toTheme: 'light' | 'dark' }
    >(null);

    const timers = useRef<number[]>([]);

    useEffect(() => {
        return () => {
            timers.current.forEach(clearTimeout);
            timers.current = [];
        };
    }, []);

    const handleClick = (e: React.MouseEvent<HTMLButtonElement>) => {
        if (overlay) return;

        const rect = e.currentTarget.getBoundingClientRect();
        const x = rect.left + rect.width / 2;
        const y = rect.top + rect.height / 2;

        // Радиус до самого дальнего угла — чтобы круг накрыл весь экран
        const dx = Math.max(x, window.innerWidth - x);
        const dy = Math.max(y, window.innerHeight - y);
        const radius = Math.hypot(dx, dy);

        const nextTheme = theme === 'light' ? 'dark' : 'light';

        timers.current.forEach(clearTimeout);
        timers.current = [];

        setOverlay({ x, y, radius, toTheme: nextTheme });

        // Тема меняется в момент, когда круг уже полностью накрыл экран (~500ms)
        timers.current.push(
            window.setTimeout(() => {
                setTheme(nextTheme);
            }, 500)
        );

        // Убираем overlay после завершения анимации
        timers.current.push(
            window.setTimeout(() => {
                setOverlay(null);
                timers.current = [];
            }, 1100)
        );
    };

    return (
        <>
            <button
                className="ThemeSwap"
                onClick={handleClick}
                aria-label="Переключить тему"
                disabled={!!overlay}
            >
                <img src={theme === 'light' ? sun : moon} alt="" />
            </button>

            {overlay && (
                <span
                    className={`theme-overlay to-${overlay.toTheme}`}
                    style={{
                        // Позиционируем круг относительно точки клика
                        left: overlay.x - overlay.radius,
                        top: overlay.y - overlay.radius,
                        width: overlay.radius * 2,
                        height: overlay.radius * 2,
                    }}
                />
            )}
        </>
    );
};

export default ThemeSwap;