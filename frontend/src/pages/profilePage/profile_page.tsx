import { useState } from 'react';
import { useAuth } from '../../context/AuthContext';
import { ProfileCardSkeleton } from '../../components/skeleton';
import '../../App.css';
import { CircleLoader } from 'react-spinners';
import ProfileCard from './profileCard'
import { ChangePasswordForm } from './ChangePasswordForm';
import EditAvatar from './EditAvatar';
import LogoutButton from "./LogoutButton";
import EditNameForm from "./EditNameForm";
import EmptyState from '../../components/EmptyState';


const ProfilePage = () => {

  const { user, loading, openModal } = useAuth();
  const [error, setError] = useState('');
  const [isPasswordOpen, setIsPasswordOpen] = useState(false);
  const [openAvatar, setOpenAvatar] = useState(false);
  const [isNameOpen, setIsNameOpen] = useState(false);


  const hadleOpenAvatarImage = () => {
    setOpenAvatar(!openAvatar);
  }






  return (

    <div className="ProfilePage">

      {loading ? (
        <ProfileCardSkeleton />
      ) : !user ? (
        <EmptyState
          icon="🔒"
          title="Вы не авторизованы"
          description="Войдите в аккаунт, чтобы увидеть свой профиль."
          action={
            <button className="Button" onClick={openModal}>
              Войти
            </button>
          }
        />
      ) : (
        <ProfileCard />
      )}


      <div>
        <button className='Button' onClick={hadleOpenAvatarImage}> Сменить аватарку</button>
        {openAvatar &&
          <EditAvatar />
        }
      </div>
      <div>
        <button className='Button' onClick={() => setIsNameOpen(true)}>
          Сменить имя
        </button>
        <EditNameForm isOpen={isNameOpen} onClose={() => setIsNameOpen(false)} />
      </div>

      <div>
        <button className='Button' onClick={() => setIsPasswordOpen(true)}>
          Сменить пароль
        </button>
        <ChangePasswordForm isOpen={isPasswordOpen} onClose={() => setIsPasswordOpen(false)} />
      </div>


      <div>
        {error && <div className="error-message">{error}</div>}

      </div>
    </div>
  );
};

export default ProfilePage;