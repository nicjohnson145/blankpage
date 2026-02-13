import {StrictMode, lazy} from 'react'
import {createRoot} from 'react-dom/client'
import {ThemeProvider, createTheme} from '@mui/material/styles';
import {createBrowserRouter} from "react-router";
import {RouterProvider} from "react-router/dom";
import {authMiddleware} from './auth/auth';
import {
    QueryClient,
    QueryClientProvider,
} from '@tanstack/react-query'

const CssBaseline = lazy(() => import("@mui/material/CssBaseline"))
const Login = lazy(() => import("./pages/Login").then(module => ({default: module.Login})))
const AllBooks = lazy(() => import("./pages/AllBooks").then(module => ({default: module.AllBooks})))
const Layout = lazy(() => import("./layouts/Layout").then(module => ({default: module.Layout})))
const SnackbarProvider = lazy(() => import("notistack").then(module => ({default: module.SnackbarProvider})))
const Edit = lazy(() => import("./pages/Edit").then(module => ({default: module.Edit})))

if (process.env.NODE_ENV === 'production') {
    console.debug = () => {}
}

const darkTheme = createTheme({
    palette: {
        mode: 'dark',
    },
});

const router = createBrowserRouter([
    {
        path: "/login",
        Component: Login,
    },
    {
        Component: Layout,
        middleware: [authMiddleware],
        children: [
            {
                index: true,
                Component: AllBooks,
            },
            {
                path: "/edit/:book_id?",
                Component: Edit,
            },
            //{
            //    path: "/add",
            //    Component: Edit,
            //},
        ],
    },
]);

const queryClient = new QueryClient();

createRoot(document.getElementById('root') as HTMLElement).render(
    <StrictMode>
        <ThemeProvider theme={darkTheme}>
            <SnackbarProvider>
                <QueryClientProvider client={queryClient}>
                    <CssBaseline />
                    <RouterProvider router={router} />
                </QueryClientProvider>
            </SnackbarProvider>
        </ThemeProvider>
    </StrictMode>,
)
