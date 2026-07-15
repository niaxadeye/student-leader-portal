import { createBrowserRouter, Navigate } from 'react-router-dom'
import { RequireAuth, RequireGuest, RequireRole } from '@/app/guards'
import { LoginPage } from '@/pages/auth/login-page'
import { ChangePasswordPage } from '@/pages/auth/change-password-page'
import { ForgotPasswordPage } from '@/pages/auth/forgot-password-page'
import { AdminLayout } from '@/pages/admin/admin-layout'
import { AdminDashboardPage } from '@/pages/admin/dashboard-page'
import { AdminContestsPage } from '@/pages/admin/contests-page'
import { AdminContestDetailPage } from '@/pages/admin/contest-detail-page'
import { ChallengeBuilderPage } from '@/pages/admin/challenge-builder-page'
import { AdminUsersPage } from '@/pages/admin/users-page'
import { OrganizersPage } from '@/pages/admin/organizers-page'
import { ContestantLayout } from '@/pages/contestant/contestant-layout'
import { DashboardPage } from '@/pages/contestant/dashboard-page'
import { ChallengeFormPage } from '@/pages/contestant/challenge-form-page'

export const router = createBrowserRouter([
  { path: '/', element: <Navigate to="/login" replace /> },
  {
    element: <RequireGuest />,
    children: [
      { path: '/login', element: <LoginPage /> },
      { path: '/forgot-password', element: <ForgotPasswordPage /> },
    ],
  },
  {
    element: <RequireAuth />,
    children: [
      { path: '/change-password', element: <ChangePasswordPage /> },
      {
        path: '/admin',
        element: <AdminLayout />,
        children: [
          { index: true, element: <AdminDashboardPage /> },
          { path: 'contests', element: <AdminContestsPage /> },
          { path: 'contests/:contestId', element: <AdminContestDetailPage /> },
          { path: 'challenges/:challengeId', element: <ChallengeBuilderPage /> },
          {
            element: <RequireRole roles={['SUPER_ADMIN', 'MEGA_ADMIN']} />,
            children: [{ path: 'users', element: <AdminUsersPage /> }],
          },
          {
            element: <RequireRole roles={['MEGA_ADMIN']} />,
            children: [{ path: 'organizers', element: <OrganizersPage /> }],
          },
        ],
      },
      {
        path: '/contestant',
        element: <ContestantLayout />,
        children: [
          { index: true, element: <DashboardPage /> },
          { path: 'challenges/:challengeId', element: <ChallengeFormPage /> },
        ],
      },
    ],
  },
  { path: '*', element: <Navigate to="/login" replace /> },
])
