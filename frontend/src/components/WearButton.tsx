import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';

const MAX = 4;
const KEY = 'button-wear';

const wearKey = (button: HTMLElement) => {
  const label = (button.textContent || button.getAttribute('aria-label') || '').replace(/\s+/g, ' ').trim();
  return `${window.location.pathname}|${label}`;
};

const readWear = (): Record<string, number> => {
  try {
    return JSON.parse(sessionStorage.getItem(KEY) || '{}');
  } catch {
    return {};
  }
};

const applySaved = () => {
  const saved = readWear();
  document.querySelectorAll<HTMLElement>('button.Button, a.Button').forEach((button) => {
    const wear = saved[wearKey(button)];
    if (!wear) return;
    button.dataset.wear = String(wear);
    if (wear >= MAX) button.dataset.wearLock = '1';
  });
};

const WearButton = () => {
  const location = useLocation();

  useEffect(() => {
    applySaved();
    const observer = new MutationObserver(applySaved);
    observer.observe(document.body, { childList: true, subtree: true });
    return () => observer.disconnect();
  }, [location.pathname]);

  useEffect(() => {
    const onClick = (event: MouseEvent) => {
      const target = event.target as HTMLElement | null;
      const button = target?.closest('button.Button, a.Button') as HTMLElement | null;
      if (!button || button.dataset.wearLock === '1') return;

      const next = Math.min(MAX, Number(button.dataset.wear || 0) + 1);
      button.dataset.wear = String(next);
      button.classList.remove('wear-hit');
      void button.offsetWidth;
      button.classList.add('wear-hit');

      const saved = readWear();
      saved[wearKey(button)] = next;
      sessionStorage.setItem(KEY, JSON.stringify(saved));

      if (next === MAX) {
        button.dataset.wearLock = '1';
        window.setTimeout(() => button.classList.remove('wear-hit'), 700);
      }
    };

    document.addEventListener('click', onClick);
    return () => document.removeEventListener('click', onClick);
  }, []);

  return null;
};

export default WearButton;
