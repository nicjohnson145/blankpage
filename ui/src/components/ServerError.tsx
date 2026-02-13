import Typography from '@mui/material/Typography';

type ServerErrorProps = {
    userMessage: string | null;
    serverMessage: string | null;
    serverCode: string | null;
};

function ServerError({ userMessage = "An error occurred" }: ServerErrorProps) {
    return (
        <>
            <Typography variant="h1">{ userMessage }</Typography>
        </>
    )
}

export {
    ServerError,
}
