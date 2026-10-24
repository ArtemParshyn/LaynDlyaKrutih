import { useEffect, useState } from 'react';
import './App.css';
import ReactModal from 'react-modal';
import { useAuth } from './context/AuthContext';
import { CircleLoader } from 'react-spinners';
import { toast } from 'sonner';

ReactModal.setAppElement('#root');

interface RegistrationModalProps {
  shouldCloseOnOverlayClick?: boolean;
  shouldCloseOnEsc?: boolean;
  isOpen: boolean;
  onClose: () => void;
  initialTab?: 'login' | 'register' | null;
}

const RegistrationModal = ({
  isOpen = false,
  onClose,
  initialTab = 'login',
  shouldCloseOnOverlayClick = false,
  shouldCloseOnEsc = false,
}: RegistrationModalProps) => {
  const { login, register: registerUser, logout, isAuthenticated } = useAuth();
  const [loading, setLoading] = useState(false);
  const [email, setEmail] = useState('');
  const [pass, setPass] = useState('');

  // Поля регистрации согласно Swagger dto.CreateUserRequest:
  // company_name (2-64), email, password (min 6), confirm_password (min 6)
  const [companyName, setCompanyName] = useState('');
  const [pass1, setPass1] = useState('');
  const [pass2, setPass2] = useState('');
  const [error, setError] = useState('');

  const [activeTab, setActiveTab] = useState<'login' | 'register' | null>(initialTab || 'login');

  const [demo, setDemo] = useState(false);

  useEffect(() => {
    if (!isOpen) return;
    setActiveTab(initialTab || 'login');
    setError('');
    setDemo(false);
    if (isAuthenticated) return;
    setEmail('');
    setPass('');
    setCompanyName('');
    setPass1('');
    setPass2('');
  }, [isOpen, initialTab, isAuthenticated]);

  useEffect(() => {
    if (!isOpen || activeTab !== 'register' || isAuthenticated) return;
    setCompanyName('');
    setEmail('');
    setPass1('');
    setPass2('');
    setDemo(true);

    const company = 'Северная верфь';
    const mail = 'example@mail.com';
    const password = 'secret1';
    let frame = 0;
    const timer = window.setInterval(() => {
      frame += 1;
      setCompanyName(company.slice(0, frame));
      setEmail(mail.slice(0, Math.max(0, frame - company.length)));
      const typedPass = password.slice(0, Math.max(0, frame - company.length - mail.length));
      setPass1(typedPass);
      setPass2(typedPass);
      if (frame >= company.length + mail.length + password.length) {
        window.clearInterval(timer);
        setDemo(false);
      }
    }, 55);

    return () => window.clearInterval(timer);
  }, [isOpen, activeTab, isAuthenticated]);

  const isLoginDisabled = !email.trim() || !pass;
  const isRegisterDisabled =
    !companyName.trim() ||
    companyName.trim().length < 2 ||
    !email.trim() ||
    !pass1 ||
    pass1.length < 6 ||
    !pass2 ||
    pass1 !== pass2;

  const handleLogin = (event: React.ChangeEvent<HTMLInputElement>) => {
    setError('');
    const target = event.target;
    if (target.name === 'email') setEmail(target.value);
    if (target.name === 'pass') setPass(target.value);
  };

  const handleRegister = (event: React.ChangeEvent<HTMLInputElement>) => {
    setDemo(false);
    setError('');
    const target = event.target;
    if (target.name === 'companyName') setCompanyName(target.value);
    if (target.name === 'email') setEmail(target.value);
    if (target.name === 'pass1') setPass1(target.value);
    if (target.name === 'pass2') setPass2(target.value);
  };

  return (
    <ReactModal
      shouldCloseOnOverlayClick={shouldCloseOnOverlayClick}
      shouldCloseOnEsc={shouldCloseOnEsc}
      className="RegistrationModal"
      overlayClassName="RegistrationModal-overlay"
      isOpen={isOpen}
      onRequestClose={onClose}
    >
      <div className="modal-tabs">
        <button
          className={`Button ${activeTab === 'login' ? 'active-tab' : ''}`}
          onClick={() => {
            setActiveTab('login');
            setError('');
          }}
        >
          Вход
        </button>

        <button
          className={`Button ${activeTab === 'register' ? 'active-tab' : ''}`}
          onClick={() => {
            setActiveTab('register');
            setError('');
          }}
        >
          Регистрация
        </button>

        <button className="Button modal-close" onClick={onClose} title="Закрыть">
          ✕
        </button>
      </div>

      <div className="modal-fields">

        {activeTab === 'login' && (
          <>
            <input
              type="email"
              placeholder="Email"
              value={email}
              name="email"
              onChange={handleLogin}
              autoComplete="email"
            />
            <input
              type="password"
              placeholder="Пароль"
              value={pass}
              name="pass"
              onChange={handleLogin}
              autoComplete="current-password"
            />
            <button
              className="Button"
              disabled={isLoginDisabled || loading}
              onClick={async () => {
                setError('');
                setLoading(true);
                try {
                  await login({ email: email.trim(), password: pass });
                  toast.success('Успешный вход в аккаунт!');
                  onClose();
                } catch (err) {
                  setError(err instanceof Error ? err.message : 'Ошибка входа');
                } finally {
                  setLoading(false);
                }
              }}
            >
              {loading ? <CircleLoader color="var(--text)" size={18} /> : 'Войти'}
            </button>
          </>
        )}

        {activeTab === 'register' && (
          <>
            <input
              type="text"
              placeholder="Название компании"
              value={companyName}
              onChange={handleRegister}
              name="companyName"
              maxLength={64}
            />
            <input
              type="email"
              placeholder="Email"
              value={email}
              onChange={handleRegister}
              name="email"
              autoComplete="email"
            />
            <input
              type="password"
              placeholder="Пароль"
              value={pass1}
              onChange={handleRegister}
              name="pass1"
              autoComplete="new-password"
            />
            <input
              type="password"
              placeholder="Повторите пароль"
              value={pass2}
              onChange={handleRegister}
              name="pass2"
              autoComplete="new-password"
            />
            <button
              className="Button"
              disabled={!demo && (isRegisterDisabled || loading)}
              onClick={async () => {
                setError('');
                setLoading(true);
                try {
                  await registerUser({
                    company_name: companyName.trim(),
                    email: email.trim(),
                    password: pass1,
                    confirm_password: pass2,
                  });
                  toast.success('Успешная регистрация!');
                  onClose();
                } catch (err) {
                  setError(err instanceof Error ? err.message : 'Ошибка регистрации');
                } finally {
                  setLoading(false);
                }
              }}
            >
              {loading ? <CircleLoader color="var(--text)" size={18} /> : 'Зарегистрироваться'}
            </button>
          </>
        )}
      </div>

      {error && <div className="error-message">{error}</div>}

      {isAuthenticated && (
        <button
          className="Button"
          onClick={async () => {
            await logout();
            onClose();
            toast.success('Вы вышли из аккаунта');
          }}
        >
          Выйти из аккаунта
        </button>
      )}
    </ReactModal>
  );
};

const ProfileButton = () => {
  const { user, openModal } = useAuth();
  const label = (user?.company_name || user?.email || 'Профиль').trim();

  return (
    <button className="profileButton" onClick={openModal} title={label} aria-label={label}>
      <img className="imageProfile" src="/images/profile.png" alt="" />
    </button>
  );
};

export { ProfileButton, RegistrationModal };
