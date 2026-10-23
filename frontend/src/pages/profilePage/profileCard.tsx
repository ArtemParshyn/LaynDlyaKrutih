import { useState } from 'react';
import { useAuth } from '../../context/AuthContext';
import '../../App.css';
import LogoutButton from "./LogoutButton";
import { motion } from 'framer-motion';
import { API_URL } from '../../api/config';

const ProfileCard = () => {

  const [error, setError] = useState('');
  const { user } = useAuth();

  const formattedDate = user?.createdAt
    ? new Date(user.createdAt).toLocaleDateString('ru-RU', {
      day: 'numeric',
      month: 'long',
      year: 'numeric',
    })
    : '—';
  const avatarSrc = user?.imageUrl
    ? `${API_URL}${user.imageUrl}`
    : '/images/profile.png';
  return (


    <div className="ProfileCard">

      <img src={avatarSrc} className="ProfilePage-avatar" />

      <div className="ProfileCard-info">
        <h1>{user?.name}</h1>

        <p>{user?.email}</p>
        <p>Дата регистрации: {formattedDate}</p>

      </div>
      <LogoutButton compact />
    </div>


  );
};

export default ProfileCard;