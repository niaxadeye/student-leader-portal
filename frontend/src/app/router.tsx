import { createBrowserRouter, Navigate } from 'react-router-dom'
import {
  RequireAuth,
  RequireGuest,
  RequireParticipantAuth,
  RequireParticipantGuest,
  RequireRole,
} from '@/app/guards'

export const router = createBrowserRouter([
  { path: '/', element: <Navigate to="/login" replace /> },
  {
    element: <RequireGuest />,
    children: [
      {
        path: '/login',
        lazy: async () => ({
          Component: (await import('@/pages/auth/login-page')).LoginPage,
        }),
      },
      {
        path: '/forgot-password',
        lazy: async () => ({
          Component: (await import('@/pages/auth/forgot-password-page')).ForgotPasswordPage,
        }),
      },
    ],
  },
  {
    element: <RequireAuth />,
    children: [
      {
        path: '/change-password',
        lazy: async () => ({
          Component: (await import('@/pages/auth/change-password-page')).ChangePasswordPage,
        }),
      },
      {
        path: '/admin',
        lazy: async () => ({
          Component: (await import('@/pages/admin/admin-layout')).AdminLayout,
        }),
        children: [
          {
            index: true,
            lazy: async () => ({
              Component: (await import('@/pages/admin/dashboard-page')).AdminDashboardPage,
            }),
          },
          {
            path: 'contests',
            lazy: async () => ({
              Component: (await import('@/pages/admin/contests-page')).AdminContestsPage,
            }),
          },
          {
            path: 'contests/:contestId',
            lazy: async () => ({
              Component: (await import('@/pages/admin/contest-detail-page')).AdminContestDetailPage,
            }),
          },
          {
            path: 'contests/:contestId/lectures/:lectureId/scanner',
            lazy: async () => ({
              Component: (await import('@/pages/admin/lecture-scanner-page')).LectureScannerPage,
            }),
          },
          {
            path: 'challenges/:challengeId',
            lazy: async () => ({
              Component: (await import('@/pages/admin/challenge-builder-page'))
                .ChallengeBuilderPage,
            }),
          },
          {
            element: <RequireRole roles={['SUPER_ADMIN', 'MEGA_ADMIN']} />,
            children: [
              {
                path: 'users',
                lazy: async () => ({
                  Component: (await import('@/pages/admin/users-page')).AdminUsersPage,
                }),
              },
            ],
          },
          {
            element: <RequireRole roles={['MEGA_ADMIN']} />,
            children: [
              {
                path: 'organizers',
                lazy: async () => ({
                  Component: (await import('@/pages/admin/organizers-page')).OrganizersPage,
                }),
              },
            ],
          },
        ],
      },
      {
        path: '/contestant',
        lazy: async () => ({
          Component: (await import('@/pages/contestant/contestant-layout')).ContestantLayout,
        }),
        children: [
          {
            index: true,
            lazy: async () => ({
              Component: (await import('@/pages/contestant/dashboard-page')).DashboardPage,
            }),
          },
          {
            path: 'challenges/:challengeId',
            lazy: async () => ({
              Component: (await import('@/pages/contestant/challenge-form-page')).ChallengeFormPage,
            }),
          },
        ],
      },
    ],
  },
  {
    path: '/event/:eventSlug',
    lazy: async () => ({
      Component: (await import('@/entities/event-participant/auth-context'))
        .ParticipantSessionLayout,
    }),
    children: [
      { index: true, element: <Navigate to="me" replace /> },
      {
        element: <RequireParticipantGuest />,
        children: [
          {
            path: 'login',
            lazy: async () => ({
              Component: (await import('@/pages/event/participant-login-page'))
                .ParticipantLoginPage,
            }),
          },
        ],
      },
      {
        element: <RequireParticipantAuth />,
        children: [
          {
            lazy: async () => ({
              Component: (await import('@/pages/event/participant-layout')).ParticipantLayout,
            }),
            children: [
              {
                path: 'me',
                lazy: async () => ({
                  Component: (await import('@/pages/event/participant-me-page')).ParticipantMePage,
                }),
              },
              {
                path: 'tasks',
                lazy: async () => ({
                  Component: (await import('@/pages/event/participant-tasks-page'))
                    .ParticipantTasksPage,
                }),
              },
              {
                path: 'tasks/:taskId',
                lazy: async () => ({
                  Component: (await import('@/pages/event/participant-task-page'))
                    .ParticipantTaskPage,
                }),
              },
              {
                path: 'shop',
                lazy: async () => ({
                  Component: (await import('@/pages/event/participant-shop-page'))
                    .ParticipantShopPage,
                }),
              },
              {
                path: 'shop/:productSlug',
                lazy: async () => ({
                  Component: (await import('@/pages/event/participant-product-page'))
                    .ParticipantProductPage,
                }),
              },
              {
                path: 'orders',
                lazy: async () => ({
                  Component: (await import('@/pages/event/participant-orders-page'))
                    .ParticipantOrdersPage,
                }),
              },
              {
                path: 'me/qr',
                lazy: async () => ({
                  Component: (await import('@/pages/event/participant-qr-page')).ParticipantQRPage,
                }),
              },
            ],
          },
        ],
      },
    ],
  },
  { path: '*', element: <Navigate to="/login" replace /> },
])
