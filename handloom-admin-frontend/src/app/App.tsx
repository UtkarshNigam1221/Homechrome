import axios from 'axios';
import { useEffect } from 'react';
import { BrowserRouter } from 'react-router-dom';

import { authApi } from '@/features/auth/api';
import { useAuthStore } from '@/shared/stores/authStore';

import { Providers } from './providers';
import { AppRoutes } from './routes';

function App() {
  useEffect(() => {
    const { login, logout, setLoading } = useAuthStore.getState();
    authApi
      .getCurrentUser()
      .then((user) => login(user))
      .catch((error: unknown) => {
        if (axios.isAxiosError(error) && error.response?.status === 401) {
          logout();
        } else {
          // Network error, timeout, 5xx, etc. — don't treat as auth failure
          setLoading(false);
        }
      });
  }, []);

  return (
    <Providers>
      <BrowserRouter>
        <AppRoutes />
      </BrowserRouter>
    </Providers>
  );
}

export default App;
