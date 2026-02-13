import {type Metadata} from "../gen/blankpage/v1/struct_pb";
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Typography from '@mui/material/Typography';
import CardActionArea from '@mui/material/CardActionArea';
import MenuBookIcon from '@mui/icons-material/MenuBook';
import Box from "@mui/material/Box";
import { useNavigate } from "react-router";

type BookCardProps = {
    metadata: Metadata;
}

function BookCard({metadata}: BookCardProps) {
    const navigate = useNavigate();

    const onClick = () => {
        navigate(`/edit/${metadata.id}`)
    };

    return (
        <>
            <Card sx={{ width: 200 }}>
                <CardActionArea onClick={onClick}>
                    <Box sx={{display: 'flex', justifyContent: 'center'}}>
                        <MenuBookIcon sx={{fontSize: 100}} />
                    </Box>
                    <CardContent>
                        <Typography variant="body1" sx={{overflow: 'hidden', whiteSpace: 'nowrap', textOverflow: 'ellipsis'}}>{metadata.title}</Typography>
                        {metadata.author &&
                            <Typography variant="body2" sx={{overflow: 'hidden', whiteSpace: 'nowrap', textOverflow: 'ellipsis'}}>{metadata.author}</Typography>
                        }
                    </CardContent>
                </CardActionArea>
            </Card>
        </>
    );
}

export {
    BookCard,
}
