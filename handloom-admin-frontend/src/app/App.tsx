import { useEffect } from 'react';
import { BrowserRouter } from 'react-router-dom';

import { authApi } from '@/features/auth/api';
import { useAuthStore } from '@/shared/stores/authStore';

import { Providers } from './providers';
import { AppRoutes } from './routes';

function App() {
  useEffect(() => {
    const { login, logout } = useAuthStore.getState();
    authApi
      .getCurrentUser()
      .then((user) => login(user))
      .catch(() => logout());
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
