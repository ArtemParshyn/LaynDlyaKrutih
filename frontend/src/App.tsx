import './App.css';
import Header from './regular/header';
import Footer from './regular/footer';
import { Routes, Route } from 'react-router-dom';
import Dashboard from './pages/Dashboard'; // НАШ НОВЫЙ ДАШБОРД
import ProfilePage from './pages/profilePage/profile_page';
import SettingsPage from './pages/settings';
import ProtectedRoute from './components/protected_route';
import { useAuth } from './context/AuthContext';
import { CircleLoader } from 'react-spinners';
import { RegistrationModal } from './registration';
import NotFoundPage from './pages/NotFoundPage';
import ErrorBoundary from "./components/ErrorBoundry/ErrorBoundry";
import WearButton from './components/WearButton';

const RegistrationModalFromContext = () => {
  const { isModalOpen, closeModal } = useAuth();
  return (
      <RegistrationModal
          shouldCloseOnOverlayClick
          shouldCloseOnEsc
          isOpen={isModalOpen}
          onClose={closeModal}
      />
  );
};

const AppContent = () => {
  const { loading, isAuthenticated } = useAuth();

  if (loading) {
    return (
        <div className="App">
          <Header />
          <main className="main">
            <div className="GuestPulse" aria-hidden="true">
              <div className="skeleton skeleton-title" />
              <div className="skeleton skeleton-chart" />
              <div className="skeleton skeleton-text" />
              <div className="skeleton skeleton-text" />
            </div>
          </main>
        </div>
    );
  }

  return (
      <div className="App">
        <WearButton />
        <Header />

        <main className='main'>
          <ErrorBoundary>
            <Routes>
              <Route path='*' element={<NotFoundPage />} />

              <Route
                  path='/'
                  element={<Dashboard />}
              />

              <Route path='/profile' element={<ProtectedRoute> <ProfilePage /> </ProtectedRoute>} />
              <Route path='/settings' element={<ProtectedRoute> <SettingsPage /> </ProtectedRoute>} />
            </Routes>
          </ErrorBoundary>
          <RegistrationModalFromContext />
        </main>
        <Footer />
      </div>
  );
}

function App() {
  return <AppContent />;
}

export default App;