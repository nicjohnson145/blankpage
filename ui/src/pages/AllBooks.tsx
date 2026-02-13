import {create} from '@bufbuild/protobuf';
import {ConnectError} from "@connectrpc/connect";
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from "@mui/material/Typography";
import {keepPreviousData, useQuery} from '@tanstack/react-query';
import {useState} from "react";
import {blankPageClient} from "../api/client";
import LoadingBar from "../components/LoadingBar";
import {QUERY_KEY_ALL_BOOKS} from "../constants";
import {ListBooksRequestSchema} from "../gen/blankpage/v1/service_pb";
import {LOCAL_STORAGE_ACCESS_KEY_KEY} from '../constants';
import {CardGrid} from '../components/CardGrid';


function AllBooks() {
    const [page, setPage] = useState(0);

    const fetchBooks = async (page = 0) => {
        const headers = new Headers();
        headers.set("Authorization", localStorage.getItem(LOCAL_STORAGE_ACCESS_KEY_KEY)!);

        return await blankPageClient().listBooks(create(ListBooksRequestSchema, {
            paginationOptions: {
                page: page,
            },
        }), {headers: headers});
    };

    const {isPending, isError, error, data, isPlaceholderData} = useQuery({
        queryKey: [QUERY_KEY_ALL_BOOKS, page],
        queryFn: () => fetchBooks(page),
        placeholderData: keepPreviousData,
        retry: 1,
    });

    const previousClick = () => {
        if (page === 0) {
            return
        }
        setPage((old) => {
            return old - 1;
        });
    };

    const nextClick = () => {
        if (isPlaceholderData || !data?.hasMore) {
            return
        }
        setPage((old) => {
            return old + 1;
        });
    };

    const getSubComponent = () => {
        if (isPending) {
            return <LoadingBar />;
        }

        if (isError) {
            const connectErr = ConnectError.from(error);
            return <Typography>{connectErr.message}</Typography>
        }

        return (
            <>
                <Stack
                    direction="row"
                    spacing={2}
                    sx={{
                        justifyContent: "flex-end",
                    }}
                >
                    <Button onClick={previousClick} disabled={page == 0}>
                        Previous
                    </Button>
                    <Button onClick={nextClick} disabled={isPlaceholderData || !data.hasMore}>
                        Next
                    </Button>
                </Stack>
                <CardGrid items={data.books} />
            </>
        )
    };
    const subComponent = getSubComponent();

    return (
        <>
            <Typography variant="h3">All Books</Typography>
            <Box>
                {subComponent}
            </Box>
        </>
    )
}

export {
    AllBooks
};

