import DashboardIcon from '@mui/icons-material/Dashboard';
import Box from '@mui/material/Box';
import Drawer from '@mui/material/Drawer';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemButton from '@mui/material/ListItemButton';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import {NavLink, Outlet} from "react-router";
import {type ReactNode} from 'react';
//import AddBoxIcon from '@mui/icons-material/AddBox';

const drawerWidth = 240;

function Layout() {
    type ListItemLinkProps = {
        name: string
        icon: ReactNode,
        to: string,
    }

    function ListItemLink(props: ListItemLinkProps): ReactNode {
        return (
            <ListItem>
                <ListItemButton component={NavLink} to={props.to}>
                    <ListItemIcon>{props.icon}</ListItemIcon>
                    <ListItemText primary={props.name}></ListItemText>
                </ListItemButton>
            </ListItem>
        )
    }

    return (
        <Box sx={{display: 'flex'}}>
            <Drawer
                sx={{
                    width: drawerWidth,
                    flexShrink: 0,
                    '& .MuiDrawer-paper': {
                        width: drawerWidth,
                        boxSizing: 'border-box',
                    },
                }}
                variant="permanent"
                anchor="left"
            >
                <List>
                    <ListItemLink to="/" name="Home" icon={<DashboardIcon />} />
                </List>
                {/* 
                <List>
                    <ListItemLink to="/add" name="Upload" icon={<AddBoxIcon />} />
                </List>
                */}
            </Drawer>
            <Box component="main" sx={{flexGrow: 1, bgcolor: 'background.default', p: 3}}>
                <Outlet />
            </Box>
        </Box>
    );
}

export {
    Layout,
}
