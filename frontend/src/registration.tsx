import { useState } from 'react';
import './App.css';
import ReactModal from 'react-modal';
import { useAuth } from './context/AuthContext';
import { CircleLoader } from 'react-spinners';
import { toast } from 'sonner';


ReactModal.setAppElement('#root');

interface RegistrationModalProps {
  shouldCloseOnOverlayClick: boolean;
  shouldCloseOnEsc: boolean;
  isOpen: boolean;
  onClose: () => void;
  initialTab?: 'login' | 'register' | null;
}

const RegistrationModal = ({ isOpen = false,
  onClose,
  initialTab = null,
  shouldCloseOnOverlayClick = false,
  shouldCloseOnEsc = false,
}: RegistrationModalProps) => {


  const { login, register: registerUser, logout, isAuthenticated } = useAuth();
  const [loading, setLoading] = useState(false);
  const [email, setEmail] = useState('');
  const [name, setName] = useState('');
  const [pass, setPass] = useState('');
  // zustand ||  MobX
  const [pass1, setPass1] = useState('');
  const [pass2, setPass2] = useState('');
  const isLoginDisabled = !email || !pass;
  const isRegisterDisabled = !name || !email || !pass1 || !pass2 || pass1 !== pass2;
  const [error, setError] = useState('')


  const [activeTab, setActiveTab] = useState<'login' | 'register' | null>(initialTab);

  const handleLogin = (event: React.ChangeEvent<HTMLInputElement>) => {
    setError('')
    const target = event.target
    const name = target.name

    if (name === 'email') setEmail(target.value)
    if (name === 'pass') setPass(target.value)
  }
  const handleRegister = (event: React.ChangeEvent<HTMLInputElement>) => {
    setError('')
    const target = event.target
    const name = target.name

    if (name === 'name') setName(target.value)
    if (name === 'pass1') setPass1(target.value)
    if (name === 'pass2') setPass2(target.value)
    if (name === 'email') setEmail(target.value)


  }




  return (

    <ReactModal
      shouldCloseOnOverlayClick={shouldCloseOnOverlayClick}
      shouldCloseOnEsc={shouldCloseOnEsc}
      className='RegistrationModal'
      overlayClassName='RegistrationModal-overlay'
      isOpen={isOpen}
      onRequestClose={onClose}>
      <div className="modal-tabs">

        <button className='Button' onClick={() => setActiveTab('login')}>Вход

        </button>


        <button className='Button' onClick={() => setActiveTab('register')}>Регистрация

        </button>
        <button className='Button'
          onClick={() => onClose()}>

          ❌
        </button>


        {activeTab === 'login' && (
          <>
            <input type="email" placeholder="Email" value={email} name='email' onChange={handleLogin} />
            <input type="password" placeholder="Пароль" value={pass} name='pass' onChange={handleLogin} />
            <button
              className='Button'
              disabled={isLoginDisabled || loading}
              onClick={async () => {
                setError('');
                setLoading(true);
                try {
                  await login({ email, password: pass });
                  toast.success('Успешный вход в аккаут)')
                  onClose();
                } catch (err) {
                  setError(err instanceof Error ? err.message : 'Ошибка входа');
                } finally {
                  setLoading(false);

                }
              }}
            >
              {loading ?
                <CircleLoader color="#fff" size={18} />
                : 'Войти'}
            </button>
          </>
        )}

        {activeTab === 'register' && (
          <>
            <input type="text" placeholder="Имя" value={name} onChange={handleRegister} name='name' />
            <input type="email" placeholder="Email" value={email} onChange={handleRegister} name='email' />
            <input type="password" placeholder="Пароль" value={pass1} onChange={handleRegister} name='pass1' />
            <input type="password" placeholder="Подтвердите пароль" value={pass2} onChange={handleRegister} name='pass2' />
            <button
              className='Button'
              disabled={isRegisterDisabled || loading}
              onClick={async () => {
                setError('');
                setLoading(true);
                try {
                  await registerUser({ name, email, password: pass1, repeatPassword: pass2 });
                  toast.success('Успешная регистрация)')
                  onClose();
                } catch (err) {
                  setError(err instanceof Error ? err.message : 'Ошибка регистрации');
                } finally {
                  setLoading(false);
                }
              }}
            >
              {loading ? <CircleLoader color="#fff" size={18} />
                : 'Зарегистрироваться'}
            </button>
          </>
        )}
      </div>
      {error && <div className="error-message">{error}</div>}
      {isAuthenticated && (
        <button className='Button' onClick={async () => {
          await logout();
          onClose();
          toast.success('Вы вышли из аккаунта)')
        }}>
          Выйти из аккаунта
        </button>
      )}
    </ReactModal>
  );
}




const ProfileButton = () => {
  const { user, openModal } = useAuth()
  const avatarSrc = user?.imageUrl
    ? `http://localhost:5000${user.imageUrl}`
    : '/images/profile.png';
  return (
    <>
      <button className='profileButton' onClick={() => openModal()}>
        <img className='imageProfile' src={avatarSrc} alt="Профиль" ></img>
      </button>
    </>
  );
};
export { ProfileButton, RegistrationModal };
