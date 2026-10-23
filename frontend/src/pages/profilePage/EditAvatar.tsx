import { useState, useRef } from 'react';
import { useAuth } from '../../context/AuthContext';
import { CircleLoader } from 'react-spinners';
import { toast } from 'sonner';
import { motion } from 'framer-motion';

const EditAvatar = () => {
  const [file, setFile] = useState<File | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const { updateAvatar } = useAuth();
  const inputRef = useRef<HTMLInputElement>(null);

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const target = event.target;
    if (target.files && target.files[0]) {
      setFile(target.files[0]);
      setError('');
    }
  };

  const handleSaveAvatar = async () => {
    if (!file) {
      setError('Ошибка, выберите файл');
      return;
    }
    setLoading(true);
    setError('');
    try {
      await updateAvatar(file);
      toast.success('Аватарка успешно изменена');
      setFile(null);

      if (inputRef.current) {
        inputRef.current.value = '';
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка загрузки');
    } finally {
      setLoading(false);
    }
  };

  return (
    <motion.div>
      <div className="ProfilePage-avatarForm">
        <label htmlFor="avatar-input" className="file-label">
          Выбрать картинку
        </label>
        <input
          ref={inputRef}
          id="avatar-input"
          type="file"
          accept="image/*"
          onChange={handleFileChange}
        />
        {file?.name && (
          <div style={{ margin: '8px 0', fontSize: '14px' }}>
            Выбранная картинка: <span className="file-name">{file.name}</span>
          </div>
        )}

        {error && <div className="error-message">{error}</div>}

        <button className="Button" onClick={handleSaveAvatar} disabled={loading || !file}>
          {loading ? <CircleLoader color="#fff" size={18} /> : 'Загрузить фото профиля'}
        </button>
      </div>
    </motion.div>
  );
};

export default EditAvatar;