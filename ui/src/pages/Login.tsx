import * as React from 'react';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import FormLabel from '@mui/material/FormLabel';
import FormControl from '@mui/material/FormControl';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import Stack from '@mui/material/Stack';
import MuiCard from '@mui/material/Card';
import {styled} from '@mui/material/styles';
import {pauthClient} from '../api/client';
import {create} from '@bufbuild/protobuf';
import {LoginRequestSchema} from '../gen/pauth/v1beta1/service_pb';
import {useSnackbar} from 'notistack';
import {ConnectError} from "@connectrpc/connect";
import { useNavigate } from "react-router";
import { LOCAL_STORAGE_ACCESS_KEY_KEY } from '../constants';

const Card = styled(MuiCard)(({theme}) => ({
    display: 'flex',
    flexDirection: 'column',
    alignSelf: 'center',
    width: '100%',
    padding: theme.spacing(4),
    gap: theme.spacing(2),
    margin: 'auto',
    [theme.breakpoints.up('sm')]: {
        maxWidth: '450px',
    },
    boxShadow:
        'hsla(220, 30%, 5%, 0.05) 0px 5px 15px 0px, hsla(220, 25%, 10%, 0.05) 0px 15px 35px -5px',
    ...theme.applyStyles('dark', {
        boxShadow:
            'hsla(220, 30%, 5%, 0.5) 0px 5px 15px 0px, hsla(220, 25%, 10%, 0.08) 0px 15px 35px -5px',
    }),
}));

const SignInContainer = styled(Stack)(({theme}) => ({
    height: 'calc((1 - var(--template-frame-height, 0)) * 100dvh)',
    minHeight: '100%',
    padding: theme.spacing(2),
    [theme.breakpoints.up('sm')]: {
        padding: theme.spacing(4),
    },
    '&::before': {
        content: '""',
        display: 'block',
        position: 'absolute',
        zIndex: -1,
        inset: 0,
    },
}));


function Login() {
    const {enqueueSnackbar} = useSnackbar();
    let navigate = useNavigate();

    const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        const data = new FormData(event.currentTarget);
        console.debug({
            email: data.get("email"),
            password: data.get("password"),
        });

        try {
            const resp = await pauthClient().login(create(LoginRequestSchema, {
                email: data.get("email")?.toString(),
                password: data.get("password")?.toString(),
            }));
            localStorage.setItem(LOCAL_STORAGE_ACCESS_KEY_KEY, resp.accessKey);
            navigate("/");
        } catch (err) {
            const connectError = ConnectError.from(err);
            enqueueSnackbar(`Error during login ${connectError.message}`, {variant: 'error', autoHideDuration: 4000})
        }
    }

    return (
        <SignInContainer>
            <Card
                variant="outlined"
            >
                <Typography variant="h4">Sign In</Typography>
                <Box
                    component="form"
                    onSubmit={handleSubmit}
                    sx={{
                        display: 'flex',
                        flexDirection: 'column',
                        width: '100%',
                        gap: 2,
                    }}
                >
                    <FormControl>
                        <FormLabel htmlFor="email">Email</FormLabel>
                        <TextField
                            id="email"
                            type="email"
                            name="email"
                            placeholder='email@example.com'
                            autoFocus
                            required
                            fullWidth
                            variant="outlined"
                        />
                    </FormControl>
                    <FormControl>
                        <FormLabel htmlFor="password">Password</FormLabel>
                        <TextField
                            id="password"
                            type="password"
                            name="password"
                            placeholder='••••••'
                            autoFocus
                            required
                            fullWidth
                            variant="outlined"
                        />
                    </FormControl>
                    <Button
                        type="submit"
                        fullWidth
                        variant='contained'
                    >
                        Sign In
                    </Button>
                </Box>
            </Card>
        </SignInContainer>
    );
}

export {
    Login,
}
