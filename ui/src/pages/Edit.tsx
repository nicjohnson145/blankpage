import {create} from '@bufbuild/protobuf';
import {ConnectError} from "@connectrpc/connect";
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import FormControl from '@mui/material/FormControl';
import FormLabel from '@mui/material/FormLabel';
import TextField from '@mui/material/TextField';
import Typography from "@mui/material/Typography";
import {useEffect, useState} from "react";
import {useForm} from "react-hook-form";
import {useParams} from "react-router";
import {blankPageClient} from "../api/client";
import {LOCAL_STORAGE_ACCESS_KEY_KEY} from '../constants';
import {ReadBookRequestSchema, UpdateBookRequestSchema} from "../gen/blankpage/v1/service_pb";
import {FieldMaskSchema} from '@bufbuild/protobuf/wkt';
import {useSnackbar} from 'notistack';
import { useNavigate } from "react-router";
import LoadingBar from "../components/LoadingBar";

type FormData = {
    title: string
    author: string | undefined
    series: string | undefined
    seriesNum: string | undefined
}

function Edit() {
    const {enqueueSnackbar} = useSnackbar();
    const navigate  = useNavigate();
    const params = useParams();
    const isEdit = params.book_id != null;

    const [isLoading, setIsLoading] = useState(isEdit);

    const {register, handleSubmit, setValue, formState: {errors}} = useForm<FormData>({});
    const pageTitle = isEdit ? "Edit Book" : "Upload Book";
    const buttonText = isEdit ? "Update" : "Upload";

    const handleEdit = async (data: FormData) => {
        try {
            const headers = new Headers();
            headers.set("Authorization", localStorage.getItem(LOCAL_STORAGE_ACCESS_KEY_KEY)!);

            let req = create(UpdateBookRequestSchema, {
                book: {
                    metadata: {
                        id: params.book_id!,
                        title: data.title,
                        author: data.author,
                        series: data.series,
                        seriesNumber: data.seriesNum === undefined ? data.seriesNum : parseFloat(data.seriesNum),
                    },
                },
                fieldMask: create(FieldMaskSchema, {
                    paths: [
                        "metadata.title",
                        "metadata.author",
                        "metadata.series",
                        "metadata.series_number",
                    ],
                }),
            });
            setIsLoading(true)
            await blankPageClient().updateBook(req, { headers: headers });
            // TODO: is this good? maybe go somewhere else?
            setIsLoading(false);
            navigate("/");
        } catch (err) {
            const connectError = ConnectError.from(err);
            enqueueSnackbar(`Error updating ${connectError.message}`, {variant: 'error', autoHideDuration: 4000})
        }
    }

    const handleAdd = async (_: FormData) => {
        enqueueSnackbar(`Add not implemented yet`, {variant: 'error', autoHideDuration: 4000})
    }

    const onSubmit = async (data: FormData) => {
        isEdit ? await handleEdit(data) : await handleAdd(data)
    };

    useEffect(() => {
        if (!isEdit) {
            return
        }

        const headers = new Headers();
        headers.set("Authorization", localStorage.getItem(LOCAL_STORAGE_ACCESS_KEY_KEY)!);

        try {
            blankPageClient().readBook(create(ReadBookRequestSchema, {
                bookId: params.book_id!,
            }), {headers: headers}).then(resp => {
                const md = resp.metadata;
                if (!md) {
                    return
                }
                setValue("title", md.title);
                if (md.author) {
                    setValue("author", md.author!)
                }
                if (md.series) {
                    setValue("series", md.series!)
                }
                if (md.seriesNumber) {
                    setValue("seriesNum", md.seriesNumber.toString())
                }
            });
        } catch (err) {
            const connectError = ConnectError.from(err);
            enqueueSnackbar(`Error fetching book data ${connectError.message}`, {variant: 'error', autoHideDuration: 4000})
        } finally {
            setIsLoading(false)
        }
    }, []);

    const getSubComponent = () => {
        const inputSx = {width: '60%', marginTop: 3};

        if (isLoading) {
            return <LoadingBar/>
        }

        return (
            <>
                <Box component="form" onSubmit={handleSubmit(onSubmit)}>
                    <FormControl sx={inputSx}>
                        <FormLabel>Title</FormLabel>
                        <TextField
                            error={!!errors.title}
                            helperText={errors.title?.message}
                            {...register("title", {required: true})}
                        />
                    </FormControl>
                    <FormControl sx={inputSx}>
                        <FormLabel>Author</FormLabel>
                        <TextField
                            error={!!errors.author}
                            helperText={errors.author?.message}
                            {...register("author")}
                        />
                    </FormControl>
                    <FormControl sx={inputSx}>
                        <FormLabel>Series</FormLabel>
                        <TextField
                            error={!!errors.series}
                            helperText={errors.series?.message}
                            {...register("series")}
                        />
                    </FormControl>
                    <FormControl sx={inputSx}>
                        <FormLabel>Series Number</FormLabel>
                        <TextField
                            error={!!errors.seriesNum}
                            helperText={errors.seriesNum?.message}
                            {...register("seriesNum")}
                        />
                    </FormControl>
                    <Button type="submit" variant="contained" sx={inputSx}>{buttonText}</Button>
                </Box>
            </>
        )
    };

    return (
        <>
            <Typography variant="h3">{pageTitle}</Typography>
            <Box>
                {getSubComponent()}
            </Box>
        </>
    )
}

export {
    Edit
};

